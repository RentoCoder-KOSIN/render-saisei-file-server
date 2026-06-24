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
func buildProjectToml(projectName, script string) string {
	return `[project]
name    = "` + projectName + `"
# version = "1.0"
# author  = "名前"

[run]
script = "` + script + `"
# args = []

# 外部パッケージが必要な場合は以下のコメントを外して記述
# [dependencies]
# packages = ["requests", "numpy"]
`
}
