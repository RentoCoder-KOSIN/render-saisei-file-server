package main

import "strings"


// buildRunBat はプロジェクトフォルダ内に同梱する run.bat の内容を生成する。
// run.bat は project.toml と同じ階層に置かれるので、カレントディレクトリ = プロジェクトフォルダ。
func buildRunBat() string {
	return `@echo off
cd /d "%~dp0"

echo.
echo  +======================================+
echo  ^|  Run Script                          ^|
echo  +======================================+
echo.

:: Check Python
python --version > nul 2>&1
if errorlevel 1 (
    py --version > nul 2>&1
    if errorlevel 1 (
        echo  [ERROR] Python not found.
        echo  Please install from https://www.python.org
        pause
        exit /b 1
    )
    set PYTHON=py
) else (
    set PYTHON=python
)

:: Check project.toml
if not exist "project.toml" (
    echo  [ERROR] project.toml not found.
    pause
    exit /b 1
)

:: Run
%PYTHON% pyrunner.py .

echo.
pause
`
}

// buildRunSh はLinux/Mac用の run.sh の内容を生成する。
func buildRunSh() string {
	return `#!/bin/bash
cd "$(dirname "$0")"

echo ""
echo " +======================================+"
echo " |  Run Script                          |"
echo " +======================================+"
echo ""

# Check Python
if command -v python3 &>/dev/null; then
    PYTHON=python3
elif command -v python &>/dev/null; then
    PYTHON=python
else
    echo " [ERROR] Python not found."
    echo " Please install from https://www.python.org"
    exit 1
fi

# Check project.toml
if [ ! -f "project.toml" ]; then
    echo " [ERROR] project.toml not found."
    exit 1
fi

# Run
$PYTHON pyrunner.py .

echo ""
read -p "Press Enter to continue..."
`
}

// detectMainScript はアップロードされたファイル一覧から実行スクリプトを探す。
// main.py があればそれを優先し、なければ最初に見つかった .py ファイルを返す。
func detectMainScript(files []struct{ Path string; Data []byte }, folderName string) string {
	first := ""
	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".py") {
			continue
		}
		name := f.Path
		// folderName/xxx.py の形式ならファイル名だけ取り出す
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		if name == "main.py" {
			return "main.py"
		}
		if first == "" {
			first = name
		}
	}
	if first != "" {
		return first
	}
	return "main.py" // fallback
}

// buildProjectToml は project.toml のテンプレート文字列を生成する。
func buildProjectToml(projectName, script string, hasReq bool, packages []string) string {
	toml := `[project]
name    = "` + projectName + `"
# version = "1.0"
# author  = "名前"

[run]
script = "` + script + `"
# args = []
`
	if hasReq {
		toml += `
[dependencies]
file = "requirements.txt"
`
	} else if len(packages) > 0 {
		quoted := make([]string, len(packages))
		for i, p := range packages {
			quoted[i] = `"` + p + `"`
		}
		toml += `
[dependencies]
packages = [` + strings.Join(quoted, ", ") + `]
`
	} else {
		toml += `
# 外部パッケージが必要な場合は以下のコメントを外して記述
# [dependencies]
# packages = ["requests", "numpy"]
`
	}
	return toml
}

// 標準ライブラリモジュール一覧（主要なもの）
var stdlibModules = map[string]bool{
	"os": true, "sys": true, "re": true, "math": true, "time": true,
	"datetime": true, "json": true, "random": true, "string": true,
	"collections": true, "itertools": true, "functools": true,
	"pathlib": true, "io": true, "abc": true, "copy": true,
	"enum": true, "typing": true, "dataclasses": true, "struct": true,
	"hashlib": true, "hmac": true, "secrets": true, "base64": true,
	"urllib": true, "http": true, "socket": true, "ssl": true,
	"threading": true, "multiprocessing": true, "subprocess": true,
	"logging": true, "unittest": true, "argparse": true, "csv": true,
	"sqlite3": true, "xml": true, "html": true, "email": true,
	"tempfile": true, "shutil": true, "glob": true, "fnmatch": true,
	"traceback": true, "inspect": true, "ast": true, "dis": true,
	"gc": true, "weakref": true, "contextlib": true, "warnings": true,
	"textwrap": true, "pprint": true, "reprlib": true, "platform": true,
	"signal": true, "queue": true, "heapq": true, "bisect": true,
	"array": true, "decimal": true, "fractions": true, "statistics": true,
	"tomllib": true, "tomli": true, "zipfile": true, "tarfile": true,
	"gzip": true, "bz2": true, "lzma": true, "zlib": true,
	"pickle": true, "shelve": true, "configparser": true,
	"tkinter": true, "turtle": true, "curses": true,
	"__future__": true, "builtins": true, "types": true,
}

// detectDependencies はファイル一覧から外部パッケージを検出する。
// requirements.txt があればそれを優先し、なければ .py ファイルを解析する。
func detectDependencies(files []struct{ Path string; Data []byte }, folderName string) (hasReq bool, packages []string) {
	// requirements.txt チェック
	for _, f := range files {
		name := f.Path
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		if name == "requirements.txt" {
			return true, nil
		}
	}

	// .py ファイルを解析して import 文から外部パッケージを検出
	seen := map[string]bool{}
	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".py") {
			continue
		}
		for _, line := range strings.Split(string(f.Data), "\n") {
			line = strings.TrimSpace(line)
			var mod string
			if strings.HasPrefix(line, "import ") {
				// import numpy, pandas など
				parts := strings.Split(strings.TrimPrefix(line, "import "), ",")
				mod = strings.TrimSpace(strings.Split(parts[0], " ")[0])
			} else if strings.HasPrefix(line, "from ") {
				// from numpy import ...
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					mod = strings.Split(parts[1], ".")[0]
				}
			}
			if mod == "" || strings.HasPrefix(mod, ".") || strings.HasPrefix(mod, "_") {
				continue
			}
			if stdlibModules[mod] {
				continue
			}
			if !seen[mod] {
				seen[mod] = true
				packages = append(packages, mod)
			}
		}
	}
	return false, packages
}
