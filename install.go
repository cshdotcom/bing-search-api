package main

// install.go 把服务安装为 systemd 单元(开机自启动、崩溃自动重启),
// 提供 CLI 子命令 install / uninstall 与 Web 端 POST /install。
//
// 安装动作:
//   1. 复制当前二进制到 /usr/local/bin/bing-search-api
//   2. 写入 /etc/systemd/system/bing-search-api.service
//   3. systemctl daemon-reload
//   4. systemctl enable  (开机自启动)
//   5. systemctl restart (立即启动/升级后重启)
//
// 非 root 调用 CLI 时自动通过 sudo 重新执行自身。

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// serviceName systemd 服务名
	serviceName = "bing-search-api"
	// installBinPath 安装目标二进制路径
	installBinPath = "/usr/local/bin/" + serviceName
	// unitPath systemd 单元文件路径
	unitPath = "/etc/systemd/system/" + serviceName + ".service"
)

// ---- 小包装(集中 os 访问,便于阅读) ----

func osGeteuid() int { return os.Geteuid() }

// osExecutableHint 当前二进制路径(用于错误提示文案)
func osExecutableHint() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "./bing-search-api"
}

// ---- 端口校验 ----

// normalizePort 校验并规范端口字符串("8080" → "8080")
func normalizePort(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("端口为空")
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("端口必须是 1~65535 的数字: %q", s)
	}
	return strconv.Itoa(n), nil
}

// ---- systemd 探测 ----

// systemdAvailable 判断当前系统是否由 systemd 管理
func systemdAvailable() bool {
	fi, err := os.Stat("/run/systemd/system")
	return err == nil && fi.IsDir()
}

// serviceActive 查询服务运行状态
func serviceActive() (bool, string) {
	out, _ := exec.Command("systemctl", "is-active", serviceName).CombinedOutput()
	return strings.TrimSpace(string(out)) == "active", strings.TrimSpace(string(out))
}

// installProbe /install?probe=1 的状态数据
func installProbe() map[string]any {
	_, unitErr := os.Stat(unitPath)
	installed := unitErr == nil
	active, state := false, ""
	if installed {
		active, state = serviceActive()
	}
	return map[string]any{
		"root":      osGeteuid() == 0,
		"systemd":   systemdAvailable(),
		"installed": installed,
		"active":    active,
		"state":     state,
		"port":      servePort,
		"service":   serviceName,
		"unit":      unitPath,
	}
}

// runCmd 执行外部命令,返回合并输出
func runCmd(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// ---- 安装/卸载核心 ----

// systemdUnit 生成 unit 文件内容
func systemdUnit(port, host string) string {
	return fmt.Sprintf(`[Unit]
Description=Bing Search API - SearXNG-compatible Bing search proxy
Documentation=https://github.com/cshdotcom/bing-search-api
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s -port %s -host %s
Restart=on-failure
RestartSec=3
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
`, installBinPath, port, host)
}

// doInstall 执行安装。start=true 时立即(重)启动服务;
// quiet=true 时不打印步骤(供 Web 端调用)。
func doInstall(port, host string, start, quiet bool) error {
	step := func(format string, a ...any) {
		if !quiet {
			fmt.Printf("  "+format+"\n", a...)
		}
	}

	if !systemdAvailable() {
		return errors.New("未检测到 systemd(容器/WSL1 环境常见):请直接前台运行本程序,或使用 Docker 部署")
	}

	// 1. 定位自身
	src, err := os.Executable()
	if err != nil {
		return fmt.Errorf("定位二进制失败: %w", err)
	}
	if resolved, rerr := filepath.EvalSymlinks(src); rerr == nil {
		src = resolved
	}
	if abs, aerr := filepath.Abs(src); aerr == nil {
		src = abs
	}

	// 2. 复制二进制到 /usr/local/bin
	if src == installBinPath {
		step("二进制已在 %s,跳过复制", installBinPath)
	} else {
		if err := copyExecutable(src, installBinPath); err != nil {
			return fmt.Errorf("复制二进制失败: %w", err)
		}
		step("二进制已安装: %s", installBinPath)
	}

	// 3. 写 unit 文件
	if err := os.WriteFile(unitPath, []byte(systemdUnit(port, host)), 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", unitPath, err)
	}
	step("服务单元已写入: %s (端口 %s)", unitPath, port)

	// 4. daemon-reload
	if out, err := runCmd("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload 失败: %s", out)
	}
	step("systemd 配置已重载")

	// 5. 开机自启动
	if out, err := runCmd("systemctl", "enable", serviceName); err != nil {
		return fmt.Errorf("设置开机自启失败: %s", out)
	}
	step("开机自启动已启用")

	// 6. 立即启动/升级重启
	if start {
		if out, err := runCmd("systemctl", "restart", serviceName); err != nil {
			return fmt.Errorf("启动服务失败: %s", out)
		}
		step("服务已启动")
	}
	return nil
}

// doUninstall 卸载服务与二进制
func doUninstall(quiet bool) error {
	step := func(format string, a ...any) {
		if !quiet {
			fmt.Printf("  "+format+"\n", a...)
		}
	}

	if _, err := os.Stat(unitPath); err != nil {
		return fmt.Errorf("未检测到已安装的服务(%s),无需卸载", unitPath)
	}

	// 先停止并取消自启(服务不存在时忽略报错)
	_, _ = runCmd("systemctl", "disable", "--now", serviceName)
	step("服务已停止,自启动已取消")

	if err := os.Remove(unitPath); err != nil {
		return fmt.Errorf("删除 %s 失败: %w", unitPath, err)
	}
	step("服务单元已删除")

	if err := os.Remove(installBinPath); err != nil {
		step("二进制删除失败(可手动清理 %s): %v", installBinPath, err)
	} else {
		step("二进制已删除: %s", installBinPath)
	}

	if out, err := runCmd("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload 失败: %s", out)
	}
	step("systemd 配置已重载")
	return nil
}

// copyExecutable 复制文件并保证可执行权限
func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		// 目录可能不存在
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		out, err = os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// ---- CLI 子命令入口(main.go 调用) ----

// runInstallCLI `bing-search-api install [-port N] [-host IP] [-no-start]`
func runInstallCLI(defPort, defHost string, rest []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	fs.Usage = func() { printSubUsage(fs, "install") }
	port := fs.String("port", defPort, "服务监听端口")
	host := fs.String("host", defHost, "服务监听地址")
	noStart := fs.Bool("no-start", false, "只安装注册,不立即启动")
	_ = fs.Parse(rest)

	portN, err := normalizePort(*port)
	if err != nil {
		fatalf("%v", err)
	}

	fmt.Println("安装 bing-search-api 为 systemd 服务(开机自启动)")

	// 非 root:自动 sudo 提权重跑
	if osGeteuid() != 0 {
		args := []string{"install", "-port", portN, "-host", *host}
		if *noStart {
			args = append(args, "-no-start")
		}
		fmt.Println("  当前非 root,通过 sudo 重新执行(可能需要输入密码)…")
		relaunchWithSudo(args) // 成功或失败都会退出
	}

	if err := doInstall(portN, *host, !*noStart, false); err != nil {
		fatalf("安装失败: %v", err)
	}

	fmt.Println()
	if active, state := serviceActive(); active {
		fmt.Printf("✓ 服务运行中 (state: %s)\n", state)
		fmt.Printf("✓ 测试界面:  http://localhost:%s/\n", portN)
		fmt.Printf("  局域网访问: http://<服务器IP>:%s/\n", portN)
	} else if *noStart {
		fmt.Println("✓ 安装完成(未启动,稍后执行 systemctl start bing-search-api)")
	} else {
		fmt.Println("⚠ 服务可能未正常启动,请检查: systemctl status bing-search-api")
		fmt.Println("  日志: journalctl -u bing-search-api -n 50")
	}
	fmt.Println()
	fmt.Println("常用管理命令:")
	fmt.Println("  systemctl status|restart|stop bing-search-api")
	fmt.Println("  journalctl -u bing-search-api -f")
	fmt.Println("  sudo bing-search-api uninstall")
}

// runUninstallCLI `bing-search-api uninstall`
func runUninstallCLI(rest []string) {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	fs.Usage = func() { printSubUsage(fs, "uninstall") }
	_ = fs.Parse(rest)

	fmt.Println("卸载 bing-search-api systemd 服务")

	if osGeteuid() != 0 {
		relaunchWithSudo([]string{"uninstall"})
	}

	if err := doUninstall(false); err != nil {
		fatalf("卸载失败: %v", err)
	}
	fmt.Println()
	fmt.Println("✓ 卸载完成")
}

// relaunchWithSudo 以 sudo 重新执行自身并透传终端,随后退出进程
func relaunchWithSudo(args []string) {
	exe, err := os.Executable()
	if err != nil {
		fatalf("定位自身二进制失败: %v", err)
	}
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		fmt.Fprintf(os.Stderr, "未安装 sudo,请手动以 root 执行:\n  sudo %s %s\n\n",
			exe, strings.Join(args, " "))
		os.Exit(1)
	}
	cmd := exec.Command(sudo, append([]string{exe}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		fatalf("sudo 执行失败: %v", err)
	}
	os.Exit(0)
}

// fatalf 打印错误并退出
func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "错误: "+format+"\n", a...)
	os.Exit(1)
}
