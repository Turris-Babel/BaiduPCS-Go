//go:build wireinject
// +build wireinject

// Package injector provides dependency injection setup using Google Wire.
package injector

import (
	"fmt"
	"os"

	"path" // Added path import
	"path/filepath"
	"runtime"
	"sort"

	"strings"
	"unicode"

	"github.com/google/wire"
	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"       // Ensure baidupcs is imported
	"github.com/qjfoidnh/BaiduPCS-Go/internal/login" // Import login package
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsfunctions/pcsdownload" // Import pcsdownload

	"github.com/peterh/liner"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand" // Re-add pcscommand import
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsupdate"  // Uncommented for RunUpdateCommand
	"github.com/qjfoidnh/BaiduPCS-Go/pcsliner"            // Ensure pcsliner is imported
	"github.com/qjfoidnh/BaiduPCS-Go/pcsliner/args"
	"github.com/qjfoidnh/BaiduPCS-Go/pcsutil"
	"github.com/qjfoidnh/BaiduPCS-Go/pcsutil/converter"
	"github.com/qjfoidnh/BaiduPCS-Go/pcsutil/escaper"
	"github.com/qjfoidnh/BaiduPCS-Go/pcsverbose"
	"github.com/qjfoidnh/BaiduPCS-Go/requester" // Use requester package
	"github.com/urfave/cli"
	"strconv" // Added strconv import
)

// --- Named Action Types ---
type QuotaAction cli.ActionFunc
type ConfigAction cli.ActionFunc
type ConfigSetAction cli.ActionFunc
type ConfigResetAction cli.ActionFunc
type LsAction cli.ActionFunc
type CdAction cli.ActionFunc
type PwdAction cli.ActionFunc
type MetaAction cli.ActionFunc
type WhoAction cli.ActionFunc
type MkdirAction cli.ActionFunc
type RmAction cli.ActionFunc
type CpAction cli.ActionFunc
type MvAction cli.ActionFunc
type LoginAction cli.ActionFunc
type DownloadAction cli.ActionFunc
type UploadAction cli.ActionFunc
type LocateAction cli.ActionFunc // Added LocateAction type
type ShareAction cli.ActionFunc
type TransferAction cli.ActionFunc
type TreeAction cli.ActionFunc
type ExportAction cli.ActionFunc

// type FixMd5Action cli.ActionFunc // Commented out for testin
// type FixMd5Action cli.ActionFunc // Commented out for testing
type RapidUploadAction cli.ActionFunc // Assuming this is separate for now
type LogoutAction cli.ActionFunc
type LoglistAction cli.ActionFunc
type ImportAction cli.ActionFunc
type UpdateAction cli.ActionFunc
type RunAction cli.ActionFunc  // Placeholder
type RunAction cli.ActionFunc // Placeholder

// TODO: Add named types for other actions like offlinedl subcommands, tool subcommands etc. if needed

// --- Command Handlers/Runners ---

// RunQuotaCommand provides the action for the 'quota' command.
// It depends on the BaiduPCS instance.
func RunQuotaCommand(pcs *baidupcs.BaiduPCS) QuotaAction { // Return named type
	return func(c *cli.Context) error {
		// Directly use the injected pcs instance
		// Call QuotaInfo() and get the returned int64 values
		quota, used, pcsError := pcs.QuotaInfo()
		if pcsError != nil {
			fmt.Printf("获取网盘配额失败: %s\n", pcsError)
			return pcsError // Return error for cli handling
		}
		// Use the returned quota and used values directly
		fmt.Printf("网盘容量: %s / %s\n", converter.ConvertFileSize(used), converter.ConvertFileSize(quota))
		return nil
	}
}

// RunConfigAction provides the action for the main 'config' command.
func RunConfigAction(cfg *pcsconfig.PCSConfig) ConfigAction { // Return named type
	return func(c *cli.Context) error {
		fmt.Printf("----\n运行 %s config set 可进行设置配置\n\n当前配置:\n", c.App.Name)
		cfg.PrintTable() // Use injected config
		return nil
	}
}

// RunConfigSetAction provides the action for the 'config set' subcommand.
func RunConfigSetAction(cfg *pcsconfig.PCSConfig) ConfigSetAction { // Return named type
	return func(c *cli.Context) error {
		if c.NumFlags() <= 0 || c.NArg() > 0 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		}
		// Use injected config (cfg)
		// TODO: Consolidate this logic if possible, maybe a helper function in pcsconfig?
		if c.IsSet("appid") {
			cfg.SetAppID(c.Int("appid"))
		}
		if c.IsSet("enable_https") {
			cfg.SetEnableHTTPS(c.Bool("enable_https"))
		}
		if c.IsSet("ignore_illegal") {
			cfg.SetIgnoreIllegal(c.Bool("ignore_illegal"))
		}
		if c.IsSet("force_login_username") {
			cfg.SetForceLogin(c.String("force_login_username"))
		}
		if c.IsSet("no_check") {
			cfg.SetNoCheck(c.Bool("no_check"))
		}
		if c.IsSet("upload_policy") {
			cfg.SetUploadPolicy(c.String("upload_policy"))
		}
		if c.IsSet("user_agent") {
			cfg.SetUserAgent(c.String("user_agent"))
		}
		if c.IsSet("pcs_ua") {
			cfg.SetPCSUA(c.String("pcs_ua"))
		}
		if c.IsSet("pcs_addr") {
			if !cfg.SETPCSAddr(c.String("pcs_addr")) {
				fmt.Println("设置 pcs_addr 错误: pcs服务器地址不合法")
				return nil // Or return an error? Original code returned nil.
			}
		}
		if c.IsSet("pan_ua") {
			cfg.SetPanUA(c.String("pan_ua"))
		}
		if c.IsSet("cache_size") {
			if err := cfg.SetCacheSizeByStr(c.String("cache_size")); err != nil {
				fmt.Printf("设置 cache_size 错误: %s\n", err)
				return nil // Or return error?
			}
		}
		if c.IsSet("max_parallel") {
			cfg.MaxParallel = c.Int("max_parallel")
		}
		if c.IsSet("max_upload_parallel") {
			cfg.MaxUploadParallel = c.Int("max_upload_parallel")
		}
		if c.IsSet("max_download_load") {
			cfg.MaxDownloadLoad = c.Int("max_download_load")
		}
		if c.IsSet("max_upload_load") {
			cfg.MaxUploadLoad = c.Int("max_upload_load")
		}
		if c.IsSet("max_download_rate") {
			if err := cfg.SetMaxDownloadRateByStr(c.String("max_download_rate")); err != nil {
				fmt.Printf("设置 max_download_rate 错误: %s\n", err)
				return nil // Or return error?
			}
		}
		if c.IsSet("max_upload_rate") {
			if err := cfg.SetMaxUploadRateByStr(c.String("max_upload_rate")); err != nil {
				fmt.Printf("设置 max_upload_rate 错误: %s\n", err)
				return nil // Or return error?
			}
		}
		if c.IsSet("savedir") {
			cfg.SaveDir = c.String("savedir")
		}
		if c.IsSet("proxy") {
			cfg.SetProxy(c.String("proxy"))
		}
		if c.IsSet("local_addrs") {
			cfg.SetLocalAddrs(c.String("local_addrs"))
		}

		err := cfg.Save() // Save using the instance
		if err != nil {
			fmt.Println(err)
			return err
		} // Return actual error
		cfg.PrintTable()
		fmt.Printf("\n保存配置成功!\n\n")
		return nil
	}
}

// RunConfigResetAction provides the action for the 'config reset' subcommand.
func RunConfigResetAction(cfg *pcsconfig.PCSConfig) ConfigResetAction { // Return named type
	return func(c *cli.Context) error {
		cfg.InitDefaultConfig() // Use injected config
		err := cfg.Save()
		if err != nil {
			fmt.Println(err)
			return err
		} // Return actual error
		cfg.PrintTable()
		fmt.Println("恢复默认配置成功")
		return nil
	}
}

// RunLsCommand provides the action for the 'ls' command.
// NOTE: This still uses pcscommand.RunLs which might rely on global state.
// Further refactoring is needed to make RunLs injectable or move its logic here.
func RunLsCommand(pcs *baidupcs.BaiduPCS) LsAction { // Return named type
	return func(c *cli.Context) error {
		// Need pcscommand import for LsOptions and RunLs
		orderOptions := &baidupcs.OrderOptions{}
		switch {
		case c.IsSet("asc"):
			orderOptions.Order = baidupcs.OrderAsc
		case c.IsSet("desc"):
			orderOptions.Order = baidupcs.OrderDesc
		default:
			orderOptions.Order = baidupcs.OrderAsc // Default to Asc as in original
		}
		switch {
		case c.IsSet("time"):
			orderOptions.By = baidupcs.OrderByTime
		case c.IsSet("name"):
			orderOptions.By = baidupcs.OrderByName
		case c.IsSet("size"):
			orderOptions.By = baidupcs.OrderBySize
		default:
			orderOptions.By = baidupcs.OrderByName // Default to Name as in original
		}

		// TODO: Refactor pcscommand.RunLs to accept pcs instance
		pcscommand.RunLs(c.Args().Get(0), &pcscommand.LsOptions{
			Total: c.Bool("l") || c.Parent().Args().Get(0) == "ll", // Original logic for ll alias
		}, orderOptions)
		return nil // Original action didn't return error explicitly
	}
}

// RunCdCommand provides the action for the 'cd' command.
// NOTE: This still uses pcscommand.RunChangeDirectory which relies on global state.
// Further refactoring is needed.
func RunCdCommand(cfg *pcsconfig.PCSConfig, pcs *baidupcs.BaiduPCS) CdAction { // Return named type
	return func(c *cli.Context) error {
		if c.NArg() == 0 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		}
		// TODO: Refactor pcscommand.RunChangeDirectory to accept cfg/pcs instances
		pcscommand.RunChangeDirectory(c.Args().Get(0), c.Bool("l"))

		// Original code had After: saveFunc. We need to save config explicitly if needed.
		// Check if RunChangeDirectory modifies config that needs saving.
		// If yes, call cfg.Save() here.
		// err := cfg.Save()
		// if err != nil {
		//	 fmt.Printf("保存配置错误: %s\n", err)
		//	 // Decide if this should be a fatal error for the command
		// }
		return nil // Original action didn't return error explicitly
	}
}

// RunPwdCommand provides the action for the 'pwd' command.
func RunPwdCommand(cfg *pcsconfig.PCSConfig) PwdAction { // Return named type
	return func(c *cli.Context) error {
		fmt.Println(cfg.ActiveUser().Workdir) // Use injected config
		return nil
	}
}

// RunMetaCommand provides the action for the 'meta' command.
// NOTE: Still uses pcscommand.RunGetMeta which relies on global state.
func RunMetaCommand(pcs *baidupcs.BaiduPCS) MetaAction { // Return named type
	return func(c *cli.Context) error {
		var (
			ca = c.Args()
			as []string
		)
		if len(ca) == 0 {
			as = []string{""}
		} else {
			as = ca
		}
		// TODO: Refactor pcscommand.RunGetMeta to accept pcs instance
		pcscommand.RunGetMeta(as...)
		return nil
	}
}

// RunWhoCommand provides the action for the 'who' command.
func RunWhoCommand(cfg *pcsconfig.PCSConfig) WhoAction { // Return named type
	return func(c *cli.Context) error {
		activeUser := cfg.ActiveUser() // Use injected config
		fmt.Printf("当前帐号 uid: %d, 用户名: %s, 性别: %s, 年龄: %.1f\n",
			activeUser.UID, activeUser.Name, activeUser.Sex, activeUser.Age)
		return nil
	}
}

// RunMkdirCommand provides the action for the 'mkdir' command.
// NOTE: Still uses pcscommand.RunMkdir which relies on global state.
func RunMkdirCommand(pcs *baidupcs.BaiduPCS) MkdirAction { // Return named type
	return func(c *cli.Context) error {
		if c.NArg() == 0 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		}
		// TODO: Refactor pcscommand.RunMkdir to accept pcs instance
		pcscommand.RunMkdir(c.Args().Get(0))
		return nil
	}
}

// RunRmCommand provides the action for the 'rm' command.
// NOTE: Still uses pcscommand.RunRemove which relies on global state.
func RunRmCommand(pcs *baidupcs.BaiduPCS) RmAction { // Return named type
	return func(c *cli.Context) error {
		if c.NArg() == 0 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		}
		// TODO: Refactor pcscommand.RunRemove to accept pcs instance
		pcscommand.RunRemove(c.Args()...)
		return nil
	}
}

// RunCpCommand provides the action for the 'cp' command.
// NOTE: Still uses pcscommand.RunCopy which relies on global state.
func RunCpCommand(pcs *baidupcs.BaiduPCS) CpAction { // Return named type
	return func(c *cli.Context) error {
		if c.NArg() <= 1 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		}
		// TODO: Refactor pcscommand.RunCopy to accept pcs instance
		pcscommand.RunCopy(c.Args()...)
		return nil
	}
}

// RunMvCommand provides the action for the 'mv' command.
// NOTE: Still uses pcscommand.RunMove which relies on global state.
func RunMvCommand(pcs *baidupcs.BaiduPCS) MvAction { // Return named type
	return func(c *cli.Context) error {
		if c.NArg() <= 1 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		}
		// TODO: Refactor pcscommand.RunMove to accept pcs instance
		pcscommand.RunMove(c.Args()...)
		return nil
	}
}

// RunLoginCommand provides the action for the 'login' command.
func RunLoginCommand(qrLogin *login.QrCodeLogin, cfg *pcsconfig.PCSConfig) LoginAction { // Return named type
	return func(c *cli.Context) error {
		// Use the injected qrLogin service
		gid, err := qrLogin.GenerateGid()
		if err != nil {
			fmt.Println("生成 GID 失败:", err)
			return err
		}
		imageUrl, sign, err := qrLogin.GetQrCode(gid)
		if err != nil {
			fmt.Println("获取二维码失败:", err)
			return err
		}
		fmt.Println("image url is:", imageUrl)
		err = qrLogin.DownloadQrCode(imageUrl)
		if err != nil {
			fmt.Println("下载二维码失败:", err)
			return err
		}
		channelV, err := qrLogin.QueryQrCode(sign, gid)
		if err != nil {
			fmt.Println("查询二维码状态失败:", err)
			return err
		}

		// Login method now uses the injected config via qrLogin service
		_, _, _, cookies, err := qrLogin.Login(channelV)
		if err != nil {
			fmt.Println("登录失败:", err)
			return err
		}

		baidu := cfg.ActiveUser() // Use injected config
		fmt.Println("百度帐号登录成功:", baidu.Name, "cookies=", cookies)

		// Save config after successful login
		err = cfg.Save()
		if err != nil {
			fmt.Printf("保存配置错误: %s\n", err)
			// Decide if this should be a fatal error for the command
		}
		return nil
	}
}

// RunDownloadCommand provides the action for the 'download' command.
// NOTE: Still uses pcscommand.RunDownload which relies on global state.
func RunDownloadCommand(pcs *baidupcs.BaiduPCS, cfg *pcsconfig.PCSConfig) DownloadAction { // Return named type
	return func(c *cli.Context) error {
		if c.NArg() == 0 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		}

		// Process saveTo path
		var saveTo string
		if c.Bool("save") {
			saveTo = "."
		} else if c.String("saveto") != "" {
			saveTo = filepath.Clean(c.String("saveto"))
		}

		// Process download mode
		var downloadMode pcsdownload.DownloadMode
		switch c.String("mode") {
		case "pcs":
			downloadMode = pcsdownload.DownloadModePCS
		case "stream":
			downloadMode = pcsdownload.DownloadModeStreaming
		case "locate":
			downloadMode = pcsdownload.DownloadModeLocate
		default:
			fmt.Println("下载方式解析失败")
			cli.ShowCommandHelp(c, c.Command.Name)
			return fmt.Errorf("无效的下载模式: %s", c.String("mode")) // Return error
		}

		// Create DownloadOptions from flags
		do := &pcscommand.DownloadOptions{
			IsTest:               c.Bool("test"),
			IsPrintStatus:        c.Bool("status"),
			IsExecutedPermission: c.Bool("x"),
			IsOverwrite:          c.Bool("ow"),
			DownloadMode:         downloadMode,
			SaveTo:               saveTo,
			Parallel:             c.Int("p"),
			Load:                 c.Int("l"),
			MaxRetry:             c.Int("retry"),
			NoCheck:              c.Bool("nocheck"),
			LinkPrefer:           c.Int("dindex"),
			ModifyMTime:          c.Bool("mtime"),
			FullPath:             c.Bool("fullpath"),
		}

		// TODO: Refactor pcscommand.RunDownload to accept pcs/cfg instances
		pcscommand.RunDownload(c.Args(), do)
		return nil
	}
}

// RunUploadCommand provides the action for the 'upload' command.
// NOTE: Still uses pcscommand.RunUpload which relies on global state.
func RunUploadCommand(pcs *baidupcs.BaiduPCS, cfg *pcsconfig.PCSConfig) UploadAction { // Return named type
	return func(c *cli.Context) error {
		if c.NArg() < 2 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		}

		subArgs := c.Args()
		// TODO: Refactor pcscommand.RunUpload to accept pcs/cfg instances
		pcscommand.RunUpload(subArgs[:c.NArg()-1], subArgs[c.NArg()-1], &pcscommand.UploadOptions{
			Parallel:      c.Int("p"),
			MaxRetry:      c.Int("retry"),
			Load:          c.Int("l"),
			NoRapidUpload: c.Bool("norapid"),
			NoSplitFile:   c.Bool("nosplit"),
			Policy:        c.String("policy"),
		})
		return nil
	}
}

// RunLocateCommand provides the action for the 'locate' command.
// NOTE: Still uses pcscommand.RunLocateDownload which relies on global state.
func RunLocateCommand(pcs *baidupcs.BaiduPCS) LocateAction { // Return named type
	return func(c *cli.Context) error {
		if c.NArg() < 1 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		}

		opt := &pcscommand.LocateDownloadOption{
			FromPan: c.Bool("pan"),
		}
		// TODO: Refactor pcscommand.RunLocateDownload to accept pcs instance
		pcscommand.RunLocateDownload(c.Args(), opt)
		return nil
	}
}

// RunShareCommand provides actions for the 'share' subcommands.
// NOTE: Still uses pcscommand functions relying on global state.
func RunShareCommand(pcs *baidupcs.BaiduPCS) ShareAction { // Return named type
	return func(c *cli.Context) error {
		// This is a placeholder for the main 'share' command action if needed.
		// Typically, subcommands handle the actual logic.
		// If 'share' itself has no action, this provider might just be for subcommands.
		// For now, let it show help like the original structure often does.
		cli.ShowCommandHelp(c, c.Command.Name)
		return nil
	}
	// TODO: Implement providers for share subcommands (list, set, cancel)
	// These would likely be separate providers injected into provideCliApp
	// Example (conceptual):
	// func RunShareListAction(pcs *baidupcs.BaiduPCS) ShareListAction { ... }
	// func RunShareSetAction(pcs *baidupcs.BaiduPCS) ShareSetAction { ... }
	// func RunShareCancelAction(pcs *baidupcs.BaiduPCS) ShareCancelAction { ... }
}

// RunTransferCommand provides actions for the 'transfer' subcommands.
// NOTE: Still uses pcscommand functions relying on global state.
func RunTransferCommand(pcs *baidupcs.BaiduPCS) TransferAction { // Return named type
	return func(c *cli.Context) error {
		// Placeholder for main 'transfer' command action.
		cli.ShowCommandHelp(c, c.Command.Name)
		return nil
	}
	// TODO: Implement providers for transfer subcommands (create, list, cancel, delete, search)
}

// RunTreeCommand provides the action for the 'tree' command.
// NOTE: Still uses pcscommand.RunTree which relies on global state.
func RunTreeCommand(pcs *baidupcs.BaiduPCS) TreeAction { // Return named type
	return func(c *cli.Context) error {
		// TODO: Add flags for depth and options to the 'tree' command in provideCliApp
		// TODO: Refactor pcscommand.RunTree to accept pcs instance instead of using global GetBaiduPCS()
		// Using default depth 0 and nil options for now
			Depth:    -1,    // Default to infinite depth as per original behavior likely
			Depth:    -1, // Default to infinite depth as per original behavior likely
			ShowFsid: false, // Default based on typical usage
		}
		// Check for flags if/when added
		// if c.IsSet("depth") { defaultOptions.Depth = c.Int("depth") }
		// if c.IsSet("fsid") { defaultOptions.ShowFsid = c.Bool("fsid") }

		pcscommand.RunTree(c.Args().Get(0), 0, defaultOptions) // Pass path, initial depth 0, and options
		return nil
	}
}

// RunExportCommand provides the action for the 'export' command.
// NOTE: Still uses pcscommand.RunExport which relies on global state.
func RunExportCommand(cfg *pcsconfig.PCSConfig) ExportAction { // Return named type
	return func(c *cli.Context) error {
		// TODO: Refactor pcscommand.RunExport to accept cfg instance instead of using global GetActiveUser() / GetBaiduPCS()
		// TODO: Add flags for other ExportOptions (RootPath, MaxRetry, Recursive, LinkFormat, StdOut)
		exportOptions := &pcscommand.ExportOptions{
			SavePath: c.String("file"), // Get save path from flag
			// Set defaults or get from other flags when added
			MaxRetry:   3,    // Example default
			Recursive:  true, // Example default (common use case)
			LinkFormat: false,
			StdOut:     false,
		}
		// if c.IsSet("recursive") { exportOptions.Recursive = c.Bool("recursive") } // Example flag check
		// ... other flags

		pcscommand.RunExport(c.Args(), exportOptions) // Pass args as []string and options struct
		return nil
	}
}

/* // Commented out RunFixMd5Command for testing
// RunFixMd5Command provides the action for the 'fixmd5' command.
// NOTE: Still uses pcscommand.RunFixMd5 which relies on global state.
func RunFixMd5Command() FixMd5Action { // Removed pcs dependency as the underlying func uses global GetBaiduPCS()
	return func(c *cli.Context) error {
		if c.NArg() == 0 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		}
		// TODO: Refactor pcscommand.RunFixMd5 to accept pcs instance
		pcscommand.RunFixMd5(c.Args()...)
		return nil
	}
}
*/

// RunLogoutCommand provides the action for the 'logout' command.
func RunLogoutCommand(cfg *pcsconfig.PCSConfig) LogoutAction {
	return func(c *cli.Context) error {
		if c.NArg() > 1 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return fmt.Errorf("logout takes at most one argument (UID or username)")
		}

		targetUID := uint64(0)
		targetName := ""
		if c.NArg() == 1 {
			target := c.Args().First()
			// Try parsing as UID first
			uid, err := strconv.ParseUint(target, 10, 64)
			if err == nil {
				targetUID = uid
			} else {
				targetName = target // Assume it's a username
			}
		} else {
			// Logout active user if no argument provided
			targetUID = cfg.ActiveUser().UID
			if targetUID == 0 {
				fmt.Println("未登录任何帐号")
				return nil
			}
			fmt.Printf("当前的登录帐号: %s\n", cfg.ActiveUser().Name)
		}

		var user *pcsconfig.Baidu
		var err error
		if targetUID != 0 {
			user, err = cfg.GetBaiduUser(&pcsconfig.BaiduBase{UID: targetUID})
		} else {
			user, err = cfg.GetBaiduUser(&pcsconfig.BaiduBase{Name: targetName})
		}

		if err != nil {
			fmt.Printf("获取用户 %s 失败: %s\n", c.Args().First(), err)
			return err
		}

		deletedUID := user.UID
		deletedName := user.Name
		_, err = cfg.DeleteUser(&pcsconfig.BaiduBase{UID: deletedUID}) // Handle two return values
		if err != nil {
			fmt.Printf("删除用户 %s 失败: %s\n", deletedName, err)
			return err
		}

		fmt.Printf("已删除用户: %s\n", deletedName)

		// Check if the active user was deleted
		if cfg.BaiduActiveUID == deletedUID {
			fmt.Println("活动用户已被删除, 尝试切换到其他用户...")
			// Switch to the first available user, if any
			if len(cfg.BaiduUserList) > 0 {
				firstUser := cfg.BaiduUserList[0]
				// Use SwitchUser instead of SetActiveUser
				_, err = cfg.SwitchUser(&pcsconfig.BaiduBase{UID: firstUser.UID})
				if err != nil {
					fmt.Printf("切换用户失败: %s\n", err)
					// Switching failed, explicitly clear active user if needed (though SwitchUser might handle this)
					// cfg.BaiduActiveUID = 0 // Might not be necessary depending on SwitchUser behavior on error
				} else {
					fmt.Printf("已切换到用户: %s\n", firstUser.Name)
				}
			} else {
				fmt.Println("没有其他用户可切换")
				cfg.BaiduActiveUID = 0 // No active user
			}
		}

		// Save the configuration changes
		err = cfg.Save()
		if err != nil {
			fmt.Printf("保存配置失败: %s\n", err)
			return err
		}
		fmt.Println("配置已保存")
		return nil
	}
}

// RunLoglistCommand provides the action for the 'loglist' command.
func RunLoglistCommand(cfg *pcsconfig.PCSConfig) LoglistAction {
	return func(c *cli.Context) error {
		// Use the String() method of the BaiduUserList slice
		fmt.Println(cfg.BaiduUserList.String())
		return nil
	}
}

// RunImportCommand provides the action for the 'import' command.
func RunImportCommand(cfg *pcsconfig.PCSConfig) ImportAction {
	return func(c *cli.Context) error {
		if c.NArg() != 1 {
			cli.ShowCommandHelp(c, c.Command.Name)
			return fmt.Errorf("import requires exactly one argument (BDUSS or file path)")
		}

		target := c.Args().First()
		var err error
		var importCount int
		// var lastImportedName string // Removed unused variable

		// Try importing from file first
		if _, statErr := os.Stat(target); statErr == nil {
			// Implement file reading logic here as ImportFromFile doesn't exist
			fileContent, readErr := os.ReadFile(target)
			if readErr != nil {
				fmt.Printf("读取文件 %s 失败: %s\n", target, readErr)
				return readErr
			}
			lines := strings.Split(string(fileContent), "\n")
			for _, line := range lines {
				bduss := strings.TrimSpace(line)
				if bduss == "" {
					continue
				}
				// Use SetupUserByBDUSS for each line (BDUSS)
				newUser, setupErr := cfg.SetupUserByBDUSS(bduss, "", "", "") // Provide empty ptoken/stoken/cookies
				if setupErr != nil {
					fmt.Printf("从文件导入 BDUSS (%s...) 失败: %s\n", converter.ShortDisplay(bduss, 10), setupErr)
					// Decide whether to continue or stop on error
					continue // Continue importing other lines
				}
				importCount++
				// lastImportedName = newUser.Name // Removed unused assignment
				fmt.Printf("从文件导入用户 %s 成功\n", newUser.Name)
			}
			if importCount == 0 {
				fmt.Printf("在文件 %s 中未找到有效的 BDUSS\n", target)
				return fmt.Errorf("no valid BDUSS found in file")
			}
			fmt.Printf("从文件 %s 共导入 %d 个用户\n", target, importCount)

		} else {
			// Assume it's a BDUSS string, use SetupUserByBDUSS
			newUser, setupErr := cfg.SetupUserByBDUSS(target, "", "", "") // Provide empty ptoken/stoken/cookies
			if setupErr != nil {
				fmt.Printf("导入 BDUSS 失败: %s\n", setupErr)
				return setupErr
			}
			importCount = 1
			// lastImportedName = newUser.Name // Removed unused assignment
			fmt.Printf("导入 BDUSS 成功, 用户: %s\n", newUser.Name)
		}

		// Save the configuration changes
		err = cfg.Save()
		if err != nil {
			fmt.Printf("保存配置失败: %s\n", err)
			return err
		}
		fmt.Println("配置已保存")
		return nil
	}
}

// RunUpdateCommand provides the action for the 'update' command.
func RunUpdateCommand() UpdateAction { // No direct dependencies needed for CheckUpdate
	return func(c *cli.Context) error {
		// Need to import internal/pcsupdate
		// Assuming CheckUpdate handles printing output itself
		// Use hardcoded version string as Version variable is not available here
		pcsupdate.CheckUpdate("v3.9.6-devel", c.Bool("y"))
		return nil
	}
}

// RunToolCommand provides the placeholder action for the 'tool' command.
func RunToolCommand() ToolAction {
	return func(c *cli.Context) error {
		// Placeholder: Show help or implement subcommands
		cli.ShowSubcommandHelp(c)
		return nil
	}
}

// RunRunCommand provides the placeholder action for the 'run' command.
func RunRunCommand() RunAction {
	return func(c *cli.Context) error {
		// Placeholder: Implement running external commands
		fmt.Println("错误: 'run' 命令尚未实现")
		return fmt.Errorf("'run' command not implemented")
	}
}

// RunRapidUploadCommand provides the action for the 'rapidupload' command.
// NOTE: Still uses pcscommand.RunRapidUpload which relies on global state.
// NOTE: The RunRapidUpload function expects MD5s and length, which are not easily obtained
//       from standard CLI args here. This command/provider likely needs refactoring.
func RunRapidUploadCommand(pcs *baidupcs.BaiduPCS) RapidUploadAction { // Removed cfg dependency
	return func(c *cli.Context) error {
		// TODO: Refactor pcscommand.RunRapidUpload to accept pcs instance instead of using global GetBaiduPCS()
		// TODO: The CLI command definition for 'rapidupload' needs to be revisited.
		//       How are contentMD5, sliceMD5, and length intended to be provided by the user?
		//       Assuming args: <targetPath> <contentMD5> <sliceMD5> <length> for now.
		if c.NArg() < 4 {
			fmt.Println("错误: rapidupload 需要 <targetPath> <contentMD5> <sliceMD5> <length> 参数")
			cli.ShowCommandHelp(c, c.Command.Name)
			return fmt.Errorf("insufficient arguments for rapidupload")
		}

		targetPath := c.Args().Get(0)
		contentMD5 := c.Args().Get(1)
		sliceMD5 := c.Args().Get(2)
		lengthStr := c.Args().Get(3)

		length, err := strconv.ParseInt(lengthStr, 10, 64) // Use strconv.ParseInt
		if err != nil {
			fmt.Printf("错误: 解析文件大小失败 '%s': %v\n", lengthStr, err)
			return err
		}

		// Policy flag is not used by RunRapidUpload function itself.
		// If policy is needed, RunRapidUpload needs modification or a different approach is required.

		pcscommand.RunRapidUpload(targetPath, contentMD5, sliceMD5, length)
		return nil
	}
}

// TODO: Add providers for other command actions like logout, offlinedl, etc.

// --- App Struct ---

// App holds the application's dependencies.
type App struct {
	CliApp *cli.App
	Config *pcsconfig.PCSConfig
	Client *requester.HTTPClient // Changed to requester.HTTPClient
	PCS    *baidupcs.BaiduPCS
	Liner  *pcsliner.PCSLiner // Corrected type name

	// Command Actions (Injected - using named types)
	QuotaAction       QuotaAction
	ConfigAction      ConfigAction
	ConfigSetAction   ConfigSetAction
	ConfigResetAction ConfigResetAction
	LsAction          LsAction
	CdAction          CdAction
	PwdAction         PwdAction
	MetaAction        MetaAction
	WhoAction         WhoAction
	MkdirAction       MkdirAction
	RmAction          RmAction
	CpAction          CpAction
	MvAction          MvAction
	LoginAction       LoginAction
	DownloadAction    DownloadAction
	UploadAction      UploadAction
	LocateAction      LocateAction // Added LocateAction field
	ShareAction       ShareAction
	TransferAction    TransferAction
	TreeAction        TreeAction
	ExportAction      ExportAction
	// FixMd5Action      FixMd5Action // Commented out for testing
	RapidUploadAction RapidUploadAction
	LogoutAction      LogoutAction
	LoglistAction     LoglistAction
	ImportAction      ImportAction
	UpdateAction      UpdateAction
	RunAction         RunAction  // Placeholder
	RunAction         RunAction // Placeholder
	// TODO: Add other action fields (e.g., for share/transfer subcommands if split)
}

// Cleanup performs necessary cleanup actions like closing resources.
func (app *App) Cleanup() {
	app.Config.Close()
	if app.Liner != nil {
		// Assuming Liner needs closing, check pcsliner implementation if needed
		// liner.Close() might be the method
	}
	fmt.Println("Cleanup finished.")
}

// provideCliApp creates and configures the main cli.App instance.
// It now depends on other providers and the injected command actions.
func provideCliApp(
	cfg *pcsconfig.PCSConfig,
	linerInstance *pcsliner.PCSLiner,
	pcs *baidupcs.BaiduPCS,
	quotaAction QuotaAction, // Inject named types
	configAction ConfigAction,
	configSetAction ConfigSetAction,
	configResetAction ConfigResetAction,
	lsAction LsAction,
	cdAction CdAction,
	pwdAction PwdAction,
	metaAction MetaAction,
	whoAction WhoAction,
	mkdirAction MkdirAction,
	rmAction RmAction,
	cpAction CpAction,
	mvAction MvAction,
	loginAction LoginAction,
	downloadAction DownloadAction,
	uploadAction UploadAction,
	locateAction LocateAction, // Inject locate action
	shareAction ShareAction, // Inject share action
	transferAction TransferAction, // Inject transfer action
	treeAction TreeAction, // Inject tree action
	exportAction ExportAction, // Inject export action
	// fixMd5Action FixMd5Action, // Commented out for testing
	rapidUploadAction RapidUploadAction, // Inject rapidupload action
	logoutAction LogoutAction, // Inject logout action
	loglistAction LoglistAction, // Inject loglist action
	importAction ImportAction, // Inject import action
	updateAction UpdateAction, // Inject update action
	toolAction ToolAction, // Inject tool action
	/* TODO: Inject other command actions */
/* TODO: Inject other command actions */
) *cli.App {
	cliApp := cli.NewApp()
	cliApp.Name = "BaiduPCS-Go"
	// TODO: Get Version from a central place
	// cliApp.Version = main.Version // Cannot access main.Version directly
	cliApp.Version = "v3.9.6-devel" // Hardcode for now, needs better solution
	cliApp.Author = "qjfoidnh/BaiduPCS-Go: https://github.com/qjfoidnh/BaiduPCS-Go"
	cliApp.Copyright = "(c) 2016-2020 iikira."
	cliApp.Usage = "百度网盘客户端 for " + runtime.GOOS + "/" + runtime.GOARCH
	cliApp.Description = `BaiduPCS-Go 使用Go语言编写的百度网盘命令行客户端, 为操作百度网盘, 提供实用功能.
具体功能, 参见 COMMANDS 列表

特色:
	网盘内列出文件和目录, 支持通配符匹配路径;
	下载网盘内文件, 支持网盘内目录 (文件夹) 下载, 支持多个文件或目录下载, 支持断点续传和高并发高速下载.

---------------------------------------------------
前往 https://github.com/qjfoidnh/BaiduPCS-Go 以获取更多帮助信息!
前往 https://github.com/qjfoidnh/BaiduPCS-Go/releases 以获取程序更新信息!
---------------------------------------------------

交流反馈:
	提交Issue: https://github.com/qjfoidnh/BaiduPCS-Go/issues
	邮箱: qjfoidnh@126.com`

	cliApp.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "verbose",
			Usage:  "启用调试",
			EnvVar: pcsverbose.EnvVerbose,
			// TODO: Refactor pcsverbose.IsVerbose to not be global or handle it differently
			// This still relies on a global variable, which is anti-pattern for DI.
			// Consider making verbose state part of the App struct or passed differently.
			Destination: &pcsverbose.IsVerbose,
		},
	}

	// Define the main action (interactive mode)
	cliApp.Action = func(c *cli.Context) {
		if c.NArg() != 0 {
			fmt.Printf("未找到命令: %s\n运行命令 %s help 获取帮助\n", c.Args().Get(0), c.App.Name)
			return
		}

		// isCli = true // TODO: Refactor this state management
		pcsverbose.Verbosef("VERBOSE: 这是一条调试信息\n\n") // TODO: Refactor pcsverbose usage

		var (
			line = linerInstance // Use the injected liner
			err  error
		)

		// History file path
		// Use pcsconfig.GetConfigDir() as cfg doesn't have ConfigDir() method
		historyFilePath := filepath.Join(pcsconfig.GetConfigDir(), "pcs_command_history.txt")

		line.History, err = pcsliner.NewLineHistory(historyFilePath)
		if err != nil {
			fmt.Printf("警告: 读取历史命令文件错误, %s\n", err)
		}

		line.ReadHistory()
		// Defer history writing and closing within the main loop or cleanup
		defer func() {
			line.DoWriteHistory()
			// linerInstance.Close() // Cleanup should handle this if needed
		}()

		// Tab completer setup
		line.State.SetCompleter(func(line string) (s []string) {
			var (
				lineArgs                   = args.Parse(line)
				numArgs                    = len(lineArgs)
				acceptCompleteFileCommands = []string{
					"cd", "cp", "download", "export", "fixmd5", "locate", "ls", "meta", "mkdir", "mv", "rapidupload", "rm", "setastoken", "share", "transfer", "tree", "upload",
				}
				closed = strings.LastIndex(line, " ") == len(line)-1
			)

			// Access commands from the current app context
			currentCliApp := c.App
			for _, cmd := range currentCliApp.Commands {
				for _, name := range cmd.Names() {
					if !strings.HasPrefix(name, line) {
						continue
					}
					s = append(s, name+" ")
				}
			}

			switch numArgs {
			case 0:
				return
			case 1:
				if !closed {
					return
				}
			}

			thisCmd := currentCliApp.Command(lineArgs[0])
			if thisCmd == nil {
				return
			}

			if !pcsutil.ContainsString(acceptCompleteFileCommands, thisCmd.FullName()) {
				return
			}

			var (
				activeUser = cfg.ActiveUser() // Use injected config
				// pcs is already injected into provideCliApp
				runeFunc    = unicode.IsSpace
				pcsRuneFunc = func(r rune) bool {
					switch r {
					case '\'', '"':
						return true
					}
					return unicode.IsSpace(r)
				}
				targetPath string
			)

			if !closed {
				targetPath = lineArgs[numArgs-1]
				escaper.EscapeStringsByRuneFunc(lineArgs[:numArgs-1], runeFunc) // 转义
			} else {
				escaper.EscapeStringsByRuneFunc(lineArgs, runeFunc)
			}

			switch {
			case targetPath == "." || strings.HasSuffix(targetPath, "/."):
				s = append(s, line+"/")
				return
			case targetPath == ".." || strings.HasSuffix(targetPath, "/.."):
				s = append(s, line+"/")
				return
			}

			var (
				targetDir string
				isAbs     = path.IsAbs(targetPath)
				isDir     = strings.LastIndex(targetPath, "/") == len(targetPath)-1
			)

			if isAbs {
				targetDir = path.Dir(targetPath)
			} else {
				targetDir = path.Join(activeUser.Workdir, targetPath)
				if !isDir {
					targetDir = path.Dir(targetDir)
				}
			}
			// Use injected pcs instance
			files, err := pcs.CacheFilesDirectoriesList(targetDir, baidupcs.DefaultOrderOptions)
			if err != nil {
				// Log or print error?
				return
			}

			for _, file := range files {
				if file == nil {
					continue
				}

				var (
					appendLine string
				)

				if !closed {
					if !strings.HasPrefix(file.Path, path.Clean(path.Join(targetDir, path.Base(targetPath)))) {
						if path.Base(targetDir) == path.Base(targetPath) {
							appendLine = strings.Join(append(lineArgs[:numArgs-1], escaper.EscapeByRuneFunc(path.Join(targetPath, file.Filename), pcsRuneFunc)), " ")
							goto handle
						}
						continue
					}
					appendLine = strings.Join(append(lineArgs[:numArgs-1], escaper.EscapeByRuneFunc(path.Clean(path.Join(path.Dir(targetPath), file.Filename)), pcsRuneFunc)), " ")
					goto handle
				}
				// 没有的情况
				appendLine = strings.Join(append(lineArgs, escaper.EscapeByRuneFunc(file.Filename, pcsRuneFunc)), " ")
			handle:
				if file.Isdir {
					s = append(s, appendLine+"/")
					continue
				}
				s = append(s, appendLine+" ")
				continue
			}

			return
		})

		fmt.Printf("提示: 方向键上下可切换历史命令.\n")
		fmt.Printf("提示: Ctrl + A / E 跳转命令 首 / 尾.\n")
		fmt.Printf("提示: 输入 help 获取帮助.\n")

		// Interactive command loop
		for {
			var (
				prompt     string
				activeUser = cfg.ActiveUser() // Use injected config
			)

			if activeUser.Name != "" {
				prompt = c.App.Name + ":" + converter.ShortDisplay(path.Base(activeUser.Workdir), 16 /*NameShortDisplayNum*/) + " " + activeUser.Name + "$ "
			} else {
				prompt = c.App.Name + " > "
			}

			commandLine, err := line.State.Prompt(prompt)
			switch err {
			case liner.ErrPromptAborted:
				// Write history before exiting interactive mode
				line.DoWriteHistory()
				return // Exit the action, which exits the app
			case nil:
				// Continue
			default:
				fmt.Println(err)
				// Write history even on error?
				line.DoWriteHistory()
				return // Exit on other errors
			}

			line.State.AppendHistory(commandLine)

			cmdArgs := args.Parse(commandLine)
			if len(cmdArgs) == 0 {
				continue
			}

			// Prepare args for running the command within the same app instance
			runArgs := []string{os.Args[0]} // App name
			runArgs = append(runArgs, cmdArgs...)

			line.Pause()
			// Run the command. Errors are typically handled by cli itself (e.g., help text).
			c.App.Run(runArgs)
			line.Resume()
		}
	}

	// Define Commands
	// TODO: Refactor command actions to accept dependencies properly.
	// This might involve creating command handler structs/functions that receive dependencies.
	cliApp.Commands = []cli.Command{
		// --- Placeholder for refactored commands ---
		// Example: login command (needs further refactoring of action)
		{
			Name:     "login",
			Usage:    "登录百度账号 (二维码扫码登录)",
			Category: "百度帐号",
			Action:   cli.ActionFunc(loginAction), // Cast named type back
		},
		// Example: config command (needs further refactoring)
		{
			Name:     "config",
			Usage:    "显示和修改程序配置项",
			Category: "配置",
			Action:   cli.ActionFunc(configAction), // Cast named type back
			Subcommands: []cli.Command{
				{
					Name:   "set",
					Usage:  "修改程序配置项",
					Action: cli.ActionFunc(configSetAction), // Cast named type back
					// Flags need to be copied/defined here for the 'set' subcommand
					Flags: []cli.Flag{
						cli.IntFlag{Name: "appid", Usage: "百度 PCS 应用ID"}, // Added Usage for clarity
						cli.StringFlag{Name: "cache_size", Usage: "下载缓存"},
						cli.IntFlag{Name: "max_parallel", Usage: "下载网络全部连接的最大并发量"},
						cli.IntFlag{Name: "max_upload_parallel", Usage: "上传网络单个连接的最大并发量"},
						cli.IntFlag{Name: "max_download_load", Usage: "同时进行下载文件的最大数量"},
						cli.IntFlag{Name: "max_upload_load", Usage: "同时进行上传文件的最大数量"},
						cli.StringFlag{Name: "max_download_rate", Usage: "限制最大下载速度, 0代表不限制"},
						cli.StringFlag{Name: "max_upload_rate", Usage: "限制最大上传速度, 0代表不限制"},
						cli.StringFlag{Name: "savedir", Usage: "下载文件的储存目录"},
						cli.BoolFlag{Name: "enable_https", Usage: "启用 https"},
						cli.BoolFlag{Name: "ignore_illegal", Usage: "忽略上传时文件名中的非法字符"},
						cli.StringFlag{Name: "force_login_username", Usage: "强制登录指定用户名"},
						cli.BoolFlag{Name: "no_check", Usage: "关闭下载文件md5校验"},
						cli.StringFlag{Name: "upload_policy", Usage: "设置上传遇到同名文件时的策略"},
						cli.StringFlag{Name: "user_agent", Usage: "浏览器标识"},
						cli.StringFlag{Name: "pcs_ua", Usage: "PCS 浏览器标识"},
						cli.StringFlag{Name: "pcs_addr", Usage: "PCS 服务器地址"},
						cli.StringFlag{Name: "pan_ua", Usage: "Pan 浏览器标识"},
						cli.StringFlag{Name: "proxy", Usage: "设置代理, 支持 http/socks5 代理"},
						cli.StringFlag{Name: "local_addrs", Usage: "设置本地网卡地址"},
					},
				},
				{
					Name:   "reset",
					Usage:  "恢复默认配置项",
					Action: cli.ActionFunc(configResetAction), // Cast named type back
				},
			},
		},
		// --- Add other commands here, refactoring their Actions ---
		// Placeholder for 'quota' command
		{
			Name:     "quota",
			Usage:    "获取网盘配额",
			Category: "百度网盘",
			Action:   cli.ActionFunc(quotaAction), // Cast named type back
		},
		// Placeholder for 'ls' command
			Name:     "ls",
			Aliases:  []string{"l", "ll"},
			Usage:    "列出目录",
			Category: "百度网盘",
			Action:   cli.ActionFunc(lsAction), // Cast named type back
			Action:    cli.ActionFunc(lsAction), // Cast named type back
			Flags: []cli.Flag{ // Flags for ls command
				cli.BoolFlag{Name: "l", Usage: "详细显示"},
				cli.BoolFlag{Name: "asc", Usage: "升序排序"},
				cli.BoolFlag{Name: "desc", Usage: "降序排序"},
				cli.BoolFlag{Name: "time", Usage: "根据时间排序"},
				cli.BoolFlag{Name: "name", Usage: "根据文件名排序"},
				cli.BoolFlag{Name: "size", Usage: "根据大小排序"},
			},
		},
		// Placeholder for 'cd' command
		{
			Name:     "cd",
			Category: "百度网盘",
			Usage:    "切换工作目录",
			Action:   cli.ActionFunc(cdAction), // Cast named type back
			Flags: []cli.Flag{ // Flags for cd command
				cli.BoolFlag{Name: "l", Usage: "切换工作目录后自动列出工作目录下的文件和目录"},
			},
		},
		// Placeholder for 'pwd' command
		{
			Name:     "pwd",
			Usage:    "输出工作目录",
			Category: "百度网盘",
			Action:   cli.ActionFunc(pwdAction), // Cast named type back
		},
		// Placeholder for 'meta' command
		{
			Name:     "meta",
			Usage:    "获取文件/目录的元信息",
			Category: "百度网盘",
			Action:   cli.ActionFunc(metaAction), // Cast named type back
		},
		// Placeholder for 'who' command
		{
			Name:     "who",
			Usage:    "获取当前帐号",
			Category: "百度帐号",
			Action:   cli.ActionFunc(whoAction), // Cast named type back
		},
		// Placeholder for 'mkdir' command
		{
			Name:     "mkdir",
			Usage:    "创建目录",
			Category: "百度网盘",
			Action:   cli.ActionFunc(mkdirAction), // Cast named type back
		},
		// Placeholder for 'rm' command
		{
			Name:     "rm",
			Usage:    "删除文件/目录",
			Category: "百度网盘",
			Action:   cli.ActionFunc(rmAction), // Cast named type back
		},
		// Placeholder for 'cp' command
		{
			Name:     "cp",
			Usage:    "拷贝文件/目录",
			Category: "百度网盘",
			Action:   cli.ActionFunc(cpAction), // Cast named type back
		},
		// Placeholder for 'mv' command
		{
			Name:     "mv",
			Usage:    "移动/重命名文件/目录",
			Category: "百度网盘",
			Action:   cli.ActionFunc(mvAction), // Cast named type back
		},
		// Placeholder for 'download' command
			Name:     "download",
			Aliases:  []string{"d"},
			Usage:    "下载文件/目录",
			Category: "百度网盘",
			Action:   cli.ActionFunc(downloadAction), // Cast named type back
			Action:    cli.ActionFunc(downloadAction), // Cast named type back
			Flags: []cli.Flag{ // Flags for download command
				cli.BoolFlag{Name: "test", Usage: "测试下载"},
				cli.BoolFlag{Name: "ow", Usage: "覆盖已存在的文件"},
				cli.BoolFlag{Name: "status", Usage: "输出所有线程的工作状态"},
				cli.BoolFlag{Name: "save", Usage: "将下载的文件直接保存到当前工作目录"},
				cli.StringFlag{Name: "saveto", Usage: "将下载的文件直接保存到指定的目录"},
				cli.BoolFlag{Name: "x", Usage: "为文件加上执行权限"},
				cli.StringFlag{Name: "mode", Usage: "下载模式 (pcs, stream, locate)", Value: "locate"},
				cli.IntFlag{Name: "p", Usage: "指定下载线程数"},
				cli.IntFlag{Name: "l", Usage: "指定同时进行下载文件的数量"},
				cli.IntFlag{Name: "retry", Usage: "下载失败最大重试次数", Value: 3}, // Value from pcsdownload.DefaultDownloadMaxRetry
				cli.BoolFlag{Name: "nocheck", Usage: "下载文件完成后不校验文件"},
				cli.BoolFlag{Name: "mtime", Usage: "将本地文件的修改时间设置为服务器上的修改时间"},
				cli.IntFlag{Name: "dindex", Usage: "使用备选下载链接中的第几个"},
				cli.BoolFlag{Name: "fullpath", Usage: "以网盘完整路径保存到本地"},
			},
		},
		// Placeholder for 'upload' command
			Name:     "upload",
			Aliases:  []string{"u"},
			Usage:    "上传文件/目录",
			Category: "百度网盘",
			Action:   cli.ActionFunc(uploadAction), // Cast named type back
			Action:    cli.ActionFunc(uploadAction), // Cast named type back
			Flags: []cli.Flag{ // Flags for upload command
				cli.IntFlag{Name: "p", Usage: "指定单个文件上传的最大线程数"},
				cli.IntFlag{Name: "retry", Usage: "上传失败最大重试次数", Value: 3}, // Value from pcscommand.DefaultUploadMaxRetry
				cli.IntFlag{Name: "l", Usage: "指定同时上传的最大文件数"},
				cli.BoolFlag{Name: "norapid", Usage: "不检测秒传"},
				cli.BoolFlag{Name: "nosplit", Usage: "禁用分片上传"},
				cli.StringFlag{Name: "policy", Usage: "对同名文件的处理策略"},
			},
		},
		// Placeholder for 'locate' command
			Name:     "locate",
			Aliases:  []string{"lt"},
			Usage:    "获取下载直链",
			Category: "百度网盘",
			Action:   cli.ActionFunc(locateAction), // Cast named type back
			Action:    cli.ActionFunc(locateAction), // Cast named type back
			Flags: []cli.Flag{ // Flags for locate command
				cli.BoolFlag{Name: "pan", Usage: "从百度网盘首页获取下载链接"},
			},
		},
		// Placeholder for 'share' command (subcommands need separate wiring)
		{
			Name:     "share",
			Usage:    "分享文件/目录",
			Category: "百度网盘",
			Action:   cli.ActionFunc(shareAction), // Cast named type back
			Subcommands: []cli.Command{
				// TODO: Define subcommands (list, set, cancel) with their own actions/providers
				{
					Name:  "list",
					Usage: "列出已分享的文件/目录",
					// Action: cli.ActionFunc(shareListAction), // Needs provider
				},
				{
					Name:  "set",
					Usage: "分享文件/目录",
					// Action: cli.ActionFunc(shareSetAction), // Needs provider
					Flags: []cli.Flag{
						cli.IntFlag{Name: "day", Usage: "过期时间 (天), 0 为永久"},
						cli.StringFlag{Name: "pwd", Usage: "提取密码, 留空则随机生成"},
					},
				},
				{
					Name:  "cancel",
					Usage: "取消分享",
					// Action: cli.ActionFunc(shareCancelAction), // Needs provider
				},
			},
		},
		// Placeholder for 'transfer' command (subcommands need separate wiring)
		{
			Name:     "transfer",
			Usage:    "离线下载",
			Category: "百度网盘",
			Action:   cli.ActionFunc(transferAction), // Cast named type back
			Subcommands: []cli.Command{
				// TODO: Define subcommands (create, list, cancel, delete, search)
				{
					Name:  "create",
					Usage: "添加离线下载任务",
					// Action: cli.ActionFunc(transferCreateAction),
					Flags: []cli.Flag{
						cli.BoolFlag{Name: "path", Usage: "将下载链接视为文件路径"},
					},
				},
				{
					Name:  "list",
					Usage: "列出离线下载任务",
					// Action: cli.ActionFunc(transferListAction),
				},
				{
					Name:  "cancel",
					Usage: "取消离线下载任务",
					// Action: cli.ActionFunc(transferCancelAction),
				},
				{
					Name:  "delete",
					Usage: "删除离线下载任务",
					// Action: cli.ActionFunc(transferDeleteAction),
				},
				{
					Name:  "search",
					Usage: "搜索离线下载任务",
					// Action: cli.ActionFunc(transferSearchAction),
				},
			},
		},
		// Placeholder for 'tree' command
		{
			Name:     "tree",
			Usage:    "树形图列出目录",
			Category: "百度网盘",
			Action:   cli.ActionFunc(treeAction), // Cast named type back
		},
		// Placeholder for 'export' command
		{
			Name:     "export",
			Usage:    "导出当前帐号的所有百度ID, BDUSS, PTOKEN, STOKEN",
			Category: "配置",
			Action:   cli.ActionFunc(exportAction), // Cast named type back
			Flags: []cli.Flag{
				cli.StringFlag{Name: "file", Usage: "导出路径"},
			},
		},
		/* // Commented out fixmd5 command for testing
		// Placeholder for 'fixmd5' command
		{
			Name:     "fixmd5",
			Usage:    "修复文件md5",
			Category: "百度网盘",
			Action:   cli.ActionFunc(fixMd5Action), // Cast named type back
		},
		*/
		// Placeholder for 'rapidupload' command
			Name:     "rapidupload",
			Aliases:  []string{"ru"},
			Usage:    "手动秒传文件",
			Category: "百度网盘",
			Action:   cli.ActionFunc(rapidUploadAction), // Cast named type back
			Action:    cli.ActionFunc(rapidUploadAction), // Cast named type back
			// Flags: []cli.Flag{
			//	// Policy flag removed as RunRapidUpload doesn't use it directly
			//	// cli.StringFlag{Name: "policy", Usage: "对同名文件的处理策略"},
			// },
			// Usage updated to reflect required args (Removed duplicate Usage field)
		},
		// Placeholder for 'logout' command
		{
			Name:     "logout",
			Usage:    "退出当前登录的百度帐号, 或指定用户名的帐号",
			Category: "百度帐号",
			Action:   cli.ActionFunc(logoutAction), // Cast named type back
		},
		// Placeholder for 'loglist' command
		{
			Name:     "loglist",
			Usage:    "列出帐号列表",
			Category: "百度帐号",
			Action:   cli.ActionFunc(loglistAction), // Cast named type back
		},
		// Placeholder for 'import' command
		{
			Name:     "import",
			Usage:    "导入百度帐号 (BDUSS 或 文件路径)",
			Category: "百度帐号",
			Action:   cli.ActionFunc(importAction), // Cast named type back
		},
		// Placeholder for 'update' command
		{
			Name:     "update",
			Usage:    "检测程序更新",
			Category: "其他",
			Action:   cli.ActionFunc(updateAction), // Cast named type back
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "y", Usage: "确认更新"},
			},
		},
		// Placeholder for 'tool' command
			Name:        "tool",
			Usage:       "实用工具",
			Category:    "其他",
			Action:      cli.ActionFunc(toolAction), // Cast named type back
			Action:   cli.ActionFunc(toolAction), // Cast named type back
			Subcommands: []cli.Command{
				// TODO: Define subcommands (e.g., checksum, crypto)
			},
		},
		// Placeholder for 'run' command
		{
			Name:     "run",
			Usage:    "执行系统命令 (未实现)",
			Category: "其他",
			Action:   cli.ActionFunc(runAction), // Cast named type back
		},
		// ... other commands need similar injection ...
		// TODO: Add commands like offlinedl (transfer subcommands), help, ver
	}

	sort.Sort(cli.FlagsByName(cliApp.Flags))
	sort.Sort(cli.CommandsByName(cliApp.Commands))

	return cliApp
}

// provideConfig initializes and returns the application configuration.
// It currently relies on the global pcsconfig.Config instance due to its pervasive use.
// A future refactor could make PCSConfig fully injectable.
func provideConfig() (*pcsconfig.PCSConfig, error) {
	// Ensure the config directory exists or can be created.
	// This logic might be better placed elsewhere, but needed for Init.
	configDir := pcsconfig.GetConfigDir()
	info, err := os.Stat(configDir)
	if err == nil && !info.IsDir() {
		return nil, fmt.Errorf("cannot create config directory %s: file exists", configDir)
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(configDir, 0700)
		if err != nil {
			return nil, fmt.Errorf("cannot create config directory %s: %w", configDir, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("cannot stat config directory %s: %w", configDir, err)
	}

	config := pcsconfig.Config // Use the global instance for now
	err = config.Init()
	// Handle non-fatal vs fatal errors as in the original init block
	if err != nil && err != pcsconfig.ErrConfigFileNoPermission && err != pcsconfig.ErrConfigContentsParseError {
		fmt.Fprintf(os.Stderr, "WARNING: config init error: %s\n", err)
		// Continue with potentially partially loaded config for non-fatal errors
	} else if err != nil {
		// Fatal errors prevent returning a usable config
		return nil, fmt.Errorf("config file error: %w", err)
	}
	return config, nil
}

// provideHTTPClient provides the HTTPClient instance from the config.
func provideHTTPClient(cfg *pcsconfig.PCSConfig) *requester.HTTPClient { // Changed return type
	// Use the pre-configured client from the config instance.
	return cfg.HTTPClient()
}

// provideActiveBaiduPCS provides the BaiduPCS instance for the currently active user.
func provideActiveBaiduPCS(cfg *pcsconfig.PCSConfig) *baidupcs.BaiduPCS {
	// This still relies on the global config's state for the active user.
	return cfg.ActiveUserBaiduPCS()
}

// provideLiner creates a new PCSLiner instance for the interactive shell.
func provideLiner() *pcsliner.PCSLiner { // Corrected return type
	return pcsliner.NewLiner()
}

// --- Provider Sets ---
var CommonSet = wire.NewSet(
	provideConfig,
	provideHTTPClient,
	provideActiveBaiduPCS,
	provideLiner,
	provideCliApp,   // Now uses injected actions
	RunQuotaCommand, // Add the provider for the quota command action
	RunConfigAction, // Add config action providers
	RunConfigSetAction,
	RunLsCommand,         // Add the provider for the ls command action
	RunCdCommand,         // Add the provider for the cd command action
	RunPwdCommand,        // Add the provider for the pwd command action
	RunMetaCommand,       // Add the provider for the meta command action
	RunWhoCommand,        // Add the provider for the who command action
	RunMkdirCommand,      // Add the provider for the mkdir command action
	RunRmCommand,         // Add the provider for the rm command action
	RunCpCommand,         // Add the provider for the cp command action
	RunMvCommand,         // Add the provider for the mv command action
	RunMvCommand, // Add the provider for the mv command action
	RunLoginCommand,      // Add the provider for the login command action
	RunDownloadCommand,   // Add the provider for the download command action
	RunUploadCommand,     // Add the provider for the upload command action
	RunLocateCommand,     // Add the provider for the locate command action
	RunShareCommand,      // Add the provider for the share command action
	RunTransferCommand,   // Add the provider for the transfer command action
	RunTreeCommand,       // Add the provider for the tree command action
	RunExportCommand,     // Add the provider for the export command action
	RunExportCommand, // Add the provider for the export command action
	// RunFixMd5Command, // Commented out for testing
	RunLogoutCommand,      // Add the provider for the logout command action
	RunLoglistCommand,     // Add the provider for the loglist command action
	RunImportCommand,      // Add the provider for the import command action
	RunUpdateCommand,      // Add the provider for the update command action
	RunToolCommand,        // Add the provider for the tool command action
	RunRunCommand,         // Add the provider for the run command action
	RunRunCommand, // Add the provider for the run command action
// TODO: Add providers for other command actions (e.g., share/transfer subcommands if split)
)

// --- Injector Function ---
// InitializeApp creates the main application instance with its dependencies.
// It returns the App instance, a cleanup function, and any error during initialization.
func InitializeApp() (*App, func(), error) {
	wire.Build(
		CommonSet,
		wire.Struct(new(App), "*"), // Wire will fill the fields of App based on providers
	)
	// The return values are placeholders; Wire generates the actual implementation.
	return nil, nil, nil
}

// </final_file_content>

// IMPORTANT: For any future changes to this file, use the final_file_content shown above as your reference. This content reflects the current state of the file, including any auto-formatting (e.g., if you used single quotes but the formatter converted them to double quotes). Always base your SEARCH/REPLACE operations on this final version to ensure accuracy.

// <environment_details>
// # VSCode Visible Files
// internal/injector/wire.go

// # VSCode Open Tabs
// internal/login/login.go
// main.go
// internal/injector/wire.go

// # Current Time
// 2025/4/5 下午5:42:21 (Asia/Shanghai, UTC+8:00)

// # Current Mode
// ACT MODE
// </environment_details>
