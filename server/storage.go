package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Supabase Storage の設定
var (
	supabaseURL    = getEnv("SUPABASE_URL", "")
	supabaseKey    = getEnv("SUPABASE_SERVICE_KEY", "")
	supabaseBucket = getEnv("SUPABASE_BUCKET", "uploads")
)

// useSupabase はSupabase Storageを使うかどうかを返す
func useSupabase() bool {
	return supabaseURL != "" && supabaseKey != ""
}

// storagePut はファイルをアップロードする（ローカル or Supabase）
func storagePut(objectPath string, data []byte) error {
	if useSupabase() {
		return supabasePut(objectPath, data)
	}
	// ローカルファイルシステム
	localPath := "uploads/" + objectPath
	if err := os.MkdirAll(dirOf(localPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(localPath, data, 0644)
}

// storageGet はファイルをダウンロードする（ローカル or Supabase）
func storageGet(objectPath string) ([]byte, error) {
	if useSupabase() {
		return supabaseGet(objectPath)
	}
	return os.ReadFile("uploads/" + objectPath)
}

// storageDelete はファイルを削除する（ローカル or Supabase）
func storageDelete(objectPath string) error {
	if useSupabase() {
		return supabaseDelete(objectPath)
	}
	return os.Remove("uploads/" + objectPath)
}

// ─── Supabase Storage API ───────────────────────────────────────────────────

func supabaseEndpoint(objectPath string) string {
	return fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, supabaseBucket, objectPath)
}

func supabasePut(objectPath string, data []byte) error {
	req, err := http.NewRequest("POST", supabaseEndpoint(objectPath), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("x-upsert", "true") // 上書き許可

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase upload error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func supabaseGet(objectPath string) ([]byte, error) {
	req, err := http.NewRequest("GET", supabaseEndpoint(objectPath), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+supabaseKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase download error %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

func supabaseDelete(objectPath string) error {
	req, err := http.NewRequest("DELETE", supabaseEndpoint(objectPath), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+supabaseKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase delete error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
