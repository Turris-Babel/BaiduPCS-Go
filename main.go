package main

import (
	"fmt"
	"log" // Added log import
	"os"

	"github.com/qjfoidnh/BaiduPCS-Go/internal/injector"  // Import the injector package
	_ "github.com/qjfoidnh/BaiduPCS-Go/internal/pcsinit" // Keep for potential side effects? Review later.
	// Remove imports only used by the moved/deleted code
	// "bytes"
	// "encoding/hex"
	// "encoding/json"
	// "errors"
	// "image"
	// "net/http"
	// "os/exec"
	// "path"
	// "path/filepath"
	// "regexp"
	// "sort"
	// "strconv"
	// "strings"
	// "time"
	// "unicode"
	// "github.com/makiuchi-d/gozxing"
	// "github.com/makiuchi-d/gozxing/qrcode"
	// "github.com/mattn/go-colorable"
	// "github.com/mdp/qrterminal"
	// "github.com/olekukonko/tablewriter"
	// "github.com/peterh/liner"
	// "github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	// "github.com/qjfoidnh/BaiduPCS-Go/client"
	// "github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	// "github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
	// "github.com/qjfoidnh/BaiduPCS-Go/internal/pcsfunctions/pcsdownload"
	// "github.com/qjfoidnh/BaiduPCS-Go/internal/pcsupdate"
	// "github.com/qjfoidnh/BaiduPCS-Go/pcsliner"
	// "github.com/qjfoidnh/BaiduPCS-Go/pcsliner/args"
	// "github.com/qjfoidnh/BaiduPCS-Go/pcstable"
	// "github.com/qjfoidnh/BaiduPCS-Go/pcsutil"
	// "github.com/qjfoidnh/BaiduPCS-Go/pcsutil/checksum"
	// "github.com/qjfoidnh/BaiduPCS-Go/pcsutil/converter"
	// "github.com/qjfoidnh/BaiduPCS-Go/pcsutil/escaper"
	// "github.com/qjfoidnh/BaiduPCS-Go/pcsutil/getip"
	// "github.com/qjfoidnh/BaiduPCS-Go/pcsutil/pcstime"
	// "github.com/qjfoidnh/BaiduPCS-Go/pcsverbose"
	// "github.com/robertkrimen/otto"
	// "github.com/urfave/cli"
)

const (
	// NameShortDisplayNum 文件名缩略显示长度 - Keep if needed elsewhere, otherwise remove.
	// NameShortDisplayNum = 16 // Commented out as it seems only used in moved code

	cryptoDescription = `
可用的方法 <method>:
	aes-128-ctr, aes-192-ctr, aes-256-ctr,
	aes-128-cfb, aes-192-cfb, aes-256-cfb,
	aes-128-ofb, aes-192-ofb, aes-256-ofb.

密钥 <key>:
	aes-128 对应key长度为16, aes-192 对应key长度为24, aes-256 对应key长度为32,
	如果key长度不符合, 则自动修剪key, 舍弃超出长度的部分, 长度不足的部分用'\0'填充.

GZIP <disable-gzip>:
	在文件加密之前, 启用GZIP压缩文件; 文件解密之后启用GZIP解压缩文件, 默认启用,
	如果不启用, 则无法检测文件是否解密成功, 解密文件时会保留源文件, 避免解密失败造成文件数据丢失.`
)

var (
	// Version 版本号 - Keep this, or move to a dedicated version package/variable accessed by injector
	Version = "v3.9.6-devel"

	// All other global variables removed
)

// --- QrCodeLogin struct and methods removed ---
// --- Helper functions htmlUnescape, ExtractNetdiskStoken removed ---

// init is removed as config initialization is handled by the injector.

// main 入口
func main() {
	// Initialize the application using the injector
	appInstance, cleanup, err := injector.InitializeApp()
	if err != nil {
		// Use log.Fatalf for fatal errors during initialization
		log.Fatalf("FATAL ERROR: failed to initialize application: %v\n", err)
	}
	// Ensure cleanup runs when main exits
	if cleanup != nil {
		// We need to call the actual cleanup function returned by the injector
		defer appInstance.Cleanup() // Call the Cleanup method on the App instance
	}

	// --- Old cli.App setup block removed ---

	// Now run the application using the instance provided by the injector
	if appInstance.CliApp == nil {
		// This should not happen if wire ran successfully
		log.Fatalf("FATAL ERROR: CLI App was not initialized by injector.")
	}

	// Set Version on the injected app instance if needed (alternative to hardcoding in provider)
	appInstance.CliApp.Version = Version

	// Run the CLI application
	err = appInstance.CliApp.Run(os.Args)
	if err != nil {
		// Handle potential errors from app run, though cli usually handles exits.
		// Use fmt.Fprintf to stderr for errors after initialization.
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		// Decide on exit code if needed, cli might already exit.
		// os.Exit(1) // Consider if an explicit exit is needed here.
	}
}
