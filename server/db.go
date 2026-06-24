package main

// ══════════════════════════════════════════════════════════════
//  DB層
//  - SUPABASE_URL + SUPABASE_SERVICE_KEY が設定されている場合
//    → Supabase PostgREST API を使って永続化
//  - 未設定の場合
//    → JSONファイルによるローカル永続化（開発用）
// ══════════════════════════════════════════════════════════════

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ── データ構造 ────────────────────────────────────────────────

type User struct {
	Username string `json:"username"`
	Salt     string `json:"salt"`
	Hash     string `json:"hash"`
	Role     string `json:"role"`
}

type Upload struct {
	ID         int    `json:"id"`
	Filename   string `json:"filename"`
	Username   string `json:"username"`
	UploadedAt string `json:"uploaded_at"`
}

// ── インメモリキャッシュ ──────────────────────────────────────

var (
	dbMu         sync.RWMutex
	usersDB      []User
	uploadsDB    []Upload
	nextUploadID = 1
	dbDir        = "."
)

func dbPath(name string) string { return filepath.Join(dbDir, name) }

// ── パスワードハッシュ ────────────────────────────────────────

func hashPassword(salt, password string) string {
	h := sha256.New()
	h.Write([]byte(salt + ":" + password))
	return hex.EncodeToString(h.Sum(nil))
}

func generateSalt() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func checkPassword(u User, password string) bool {
	return u.Hash == hashPassword(u.Salt, password)
}

// ── Supabase PostgREST ────────────────────────────────────────

func supabaseRestURL(table string) string {
	return fmt.Sprintf("%s/rest/v1/%s", supabaseURL, table)
}

func supabaseRequest(method, url string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("supabase error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// ── 初期化 ────────────────────────────────────────────────────

func initDB() {
	if p := getEnv("DB_PATH", ""); p != "" {
		dbDir = filepath.Dir(p)
	}
	os.MkdirAll(dbDir, 0755)

	loadUsers()
	loadUploads()

	// adminが存在しない場合のみ作成
	createUserIfNotExists("admin", "admin123", "teacher")
}

// ── Users ─────────────────────────────────────────────────────

func loadUsers() {
	if useSupabase() {
		data, err := supabaseRequest("GET", supabaseRestURL("users"), nil)
		if err != nil {
			fmt.Println("loadUsers error:", err)
			return
		}
		json.Unmarshal(data, &usersDB)
		return
	}
	// ローカル
	data, err := os.ReadFile(dbPath("users.json"))
	if err != nil {
		return
	}
	json.Unmarshal(data, &usersDB)
}

func saveUsers() {
	if useSupabase() {
		return // Supabase使用時は各操作で直接APIを叩くため不要
	}
	data, _ := json.MarshalIndent(usersDB, "", "  ")
	os.WriteFile(dbPath("users.json"), data, 0644)
}

func createUserIfNotExists(username, password, role string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	for _, u := range usersDB {
		if u.Username == username {
			return
		}
	}
	salt := generateSalt()
	u := User{Username: username, Salt: salt, Hash: hashPassword(salt, password), Role: role}

	if useSupabase() {
		supabaseRequest("POST", supabaseRestURL("users"), u)
	}
	usersDB = append(usersDB, u)
	saveUsers()
}

func getRole(username string) (string, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	for _, u := range usersDB {
		if u.Username == username {
			return u.Role, nil
		}
	}
	return "", fmt.Errorf("user not found")
}

func authUser(username, password string) (string, error) {
	// 毎回Supabaseから取得（再デプロイ後も最新を参照）
	if useSupabase() {
		url := supabaseRestURL("users") + "?username=eq." + username
		data, err := supabaseRequest("GET", url, nil)
		if err != nil {
			return "", fmt.Errorf("db error: %w", err)
		}
		var users []User
		if err := json.Unmarshal(data, &users); err != nil || len(users) == 0 {
			return "", fmt.Errorf("user not found")
		}
		u := users[0]
		if !checkPassword(u, password) {
			return "", fmt.Errorf("wrong password")
		}
		// キャッシュ更新
		dbMu.Lock()
		found := false
		for i, cu := range usersDB {
			if cu.Username == username {
				usersDB[i] = u
				found = true
				break
			}
		}
		if !found {
			usersDB = append(usersDB, u)
		}
		dbMu.Unlock()
		return u.Role, nil
	}

	dbMu.RLock()
	defer dbMu.RUnlock()
	for _, u := range usersDB {
		if u.Username == username {
			if !checkPassword(u, password) {
				return "", fmt.Errorf("wrong password")
			}
			return u.Role, nil
		}
	}
	return "", fmt.Errorf("user not found")
}

func listUsers() []User {
	if useSupabase() {
		data, err := supabaseRequest("GET", supabaseRestURL("users"), nil)
		if err == nil {
			var users []User
			if json.Unmarshal(data, &users) == nil {
				sort.Slice(users, func(i, j int) bool {
					if users[i].Role != users[j].Role {
						return users[i].Role < users[j].Role
					}
					return users[i].Username < users[j].Username
				})
				return users
			}
		}
	}
	dbMu.RLock()
	defer dbMu.RUnlock()
	result := make([]User, len(usersDB))
	copy(result, usersDB)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Role != result[j].Role {
			return result[i].Role < result[j].Role
		}
		return result[i].Username < result[j].Username
	})
	return result
}

func addUser(username, password, role string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	for _, u := range usersDB {
		if u.Username == username {
			return fmt.Errorf("already exists")
		}
	}
	salt := generateSalt()
	u := User{Username: username, Salt: salt, Hash: hashPassword(salt, password), Role: role}

	if useSupabase() {
		if _, err := supabaseRequest("POST", supabaseRestURL("users"), u); err != nil {
			return err
		}
	}
	usersDB = append(usersDB, u)
	saveUsers()
	return nil
}

func deleteUser(username string) (bool, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	if useSupabase() {
		url := supabaseRestURL("users") + "?username=eq." + username
		if _, err := supabaseRequest("DELETE", url, nil); err != nil {
			return false, err
		}
	}
	for i, u := range usersDB {
		if u.Username == username {
			usersDB = append(usersDB[:i], usersDB[i+1:]...)
			saveUsers()
			return true, nil
		}
	}
	return false, nil
}

func changePassword(username, oldPassword, newPassword string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	for i, u := range usersDB {
		if u.Username == username {
			if !checkPassword(u, oldPassword) {
				return fmt.Errorf("wrong password")
			}
			salt := generateSalt()
			usersDB[i].Salt = salt
			usersDB[i].Hash = hashPassword(salt, newPassword)

			if useSupabase() {
				url := supabaseRestURL("users") + "?username=eq." + username
				supabaseRequest("PATCH", url, map[string]string{
					"salt": salt,
					"hash": usersDB[i].Hash,
				})
			}
			saveUsers()
			return nil
		}
	}
	return fmt.Errorf("user not found")
}

// ── Uploads ───────────────────────────────────────────────────

func loadUploads() {
	if useSupabase() {
		url := supabaseRestURL("uploads") + "?order=id.desc"
		data, err := supabaseRequest("GET", url, nil)
		if err != nil {
			fmt.Println("loadUploads error:", err)
			return
		}
		json.Unmarshal(data, &uploadsDB)
		for _, u := range uploadsDB {
			if u.ID >= nextUploadID {
				nextUploadID = u.ID + 1
			}
		}
		return
	}
	data, err := os.ReadFile(dbPath("uploads.json"))
	if err != nil {
		return
	}
	json.Unmarshal(data, &uploadsDB)
	for _, u := range uploadsDB {
		if u.ID >= nextUploadID {
			nextUploadID = u.ID + 1
		}
	}
}

func saveUploads() {
	if useSupabase() {
		return
	}
	data, _ := json.MarshalIndent(uploadsDB, "", "  ")
	os.WriteFile(dbPath("uploads.json"), data, 0644)
}

func insertUpload(filename, username string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	u := Upload{
		ID:         nextUploadID,
		Filename:   filename,
		Username:   username,
		UploadedAt: time.Now().In(jst()).Format("2006-01-02 15:04:05"),
	}
	if useSupabase() {
		supabaseRequest("POST", supabaseRestURL("uploads"), u)
	}
	uploadsDB = append(uploadsDB, u)
	nextUploadID++
	saveUploads()
}

func listUploads() []Upload {
	if useSupabase() {
		url := supabaseRestURL("uploads") + "?order=id.desc"
		data, err := supabaseRequest("GET", url, nil)
		if err == nil {
			var uploads []Upload
			if json.Unmarshal(data, &uploads) == nil {
				return uploads
			}
		}
	}
	dbMu.RLock()
	defer dbMu.RUnlock()
	result := make([]Upload, len(uploadsDB))
	copy(result, uploadsDB)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID > result[j].ID
	})
	return result
}

func deleteUpload(filename string) {
	dbMu.Lock()
	defer dbMu.Unlock()

	if useSupabase() {
		url := supabaseRestURL("uploads") + "?filename=eq." + filename
		supabaseRequest("DELETE", url, nil)
	}
	for i, u := range uploadsDB {
		if u.Filename == filename {
			uploadsDB = append(uploadsDB[:i], uploadsDB[i+1:]...)
			saveUploads()
			return
		}
	}
}

func jst() *time.Location {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return time.FixedZone("JST", 9*60*60)
	}
	return loc
}
