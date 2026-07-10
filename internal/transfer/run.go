package transfer

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"scpick/internal/auth"
	"scpick/internal/picker"
	"scpick/internal/remotefs"
	"scpick/internal/sshconf"
)

const manualEntryPath = "\x00manual-entry"

// RunPull drives the full interactive pull flow: host selection,
// authentication, remote file browsing, local destination browsing, and
// transfer with a printed summary. It is the entry point cmd/scpick calls
// for `scpick pull`. When recursive is true, whole directories may be
// selected during browsing and are downloaded recursively (like scp -r).
func RunPull(recursive bool) error {
	client, err := connect()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	selection, err := BrowseRemoteFiles(client, "/", recursive)
	if err != nil {
		return fmt.Errorf("select remote file(s): %w", err)
	}

	localStart, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	localDir, err := BrowseLocalDir(localStart)
	if err != nil {
		return fmt.Errorf("select local destination: %w", err)
	}

	var result Result
	if len(selection.Directories) > 0 {
		result = recursivePull(client, selection.Files, selection.Directories, localDir, DefaultConfirmOverwrite, DefaultProgressPrinter)
	} else {
		result = Pull(client, selection.Files, localDir, DefaultConfirmOverwrite, DefaultProgressPrinter)
	}
	printSummary(result)
	if len(result.Failed) > 0 {
		return fmt.Errorf("%d file(s) failed to transfer", len(result.Failed))
	}
	return nil
}

// RunPush drives the full interactive push flow: host selection,
// authentication, local file browsing, remote destination browsing, and
// transfer with a printed summary. It is the entry point cmd/scpick calls
// for `scpick push`. When recursive is true, whole directories may be
// selected during browsing and are uploaded recursively (like scp -r).
func RunPush(recursive bool) error {
	client, err := connect()
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	localStart, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	selection, err := BrowseLocalFiles(localStart, recursive)
	if err != nil {
		return fmt.Errorf("select local file(s): %w", err)
	}

	remoteDir, err := BrowseRemoteDir(client, "/")
	if err != nil {
		return fmt.Errorf("select remote destination: %w", err)
	}

	var result Result
	if len(selection.Directories) > 0 {
		result = recursivePush(client, selection.Files, selection.Directories, remoteDir, DefaultConfirmOverwrite, DefaultProgressPrinter)
	} else {
		result = Push(client, selection.Files, remoteDir, DefaultConfirmOverwrite, DefaultProgressPrinter)
	}
	printSummary(result)
	if len(result.Failed) > 0 {
		return fmt.Errorf("%d file(s) failed to transfer", len(result.Failed))
	}
	return nil
}

func connect() (*remotefs.Client, error) {
	host, err := pickHost()
	if err != nil {
		return nil, fmt.Errorf("select host: %w", err)
	}

	authChain, err := auth.NewAuthChain()
	if err != nil {
		return nil, fmt.Errorf("init auth: %w", err)
	}

	khPath, err := knownHostsPath()
	if err != nil {
		return nil, err
	}
	hostKeyCB, err := auth.HostKeyCallback(khPath, confirmHostKey)
	if err != nil {
		return nil, fmt.Errorf("known_hosts: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", host.Hostname, host.Port)
	client, err := remotefs.Dial(addr, host.User, authChain.SSHAuthMethods(host.User), hostKeyCB)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", addr, err)
	}
	return client, nil
}

// pickHost offers every Host in ~/.ssh/config plus a manual-entry option.
func pickHost() (sshconf.Host, error) {
	cfg, err := sshconf.LoadConfig()
	if err != nil {
		return sshconf.Host{}, err
	}

	hosts := cfg.Hosts()
	items := make([]picker.ListItem, 0, len(hosts)+1)
	for _, h := range hosts {
		items = append(items, picker.ListItem{
			Label: fmt.Sprintf("%s (%s@%s:%d)", h.Name, h.User, h.Hostname, h.Port),
			Path:  h.Name,
		})
	}
	items = append(items, picker.ListItem{Label: "(enter host manually)", Path: manualEntryPath})

	selected, err := picker.PickOne(items)
	if err != nil {
		return sshconf.Host{}, err
	}
	if selected.Path == manualEntryPath {
		return readManualHost()
	}
	if h := cfg.FindHost(selected.Path); h != nil {
		return *h, nil
	}
	return sshconf.Host{}, fmt.Errorf("host %q not found", selected.Path)
}

func readManualHost() (sshconf.Host, error) {
	host := sshconf.Host{Port: 22}

	hostname, err := readLine("Hostname: ")
	if err != nil {
		return sshconf.Host{}, fmt.Errorf("read hostname: %w", err)
	}
	host.Hostname = strings.TrimSpace(hostname)
	host.Name = host.Hostname

	user, err := readLine("User: ")
	if err != nil {
		return sshconf.Host{}, fmt.Errorf("read user: %w", err)
	}
	host.User = strings.TrimSpace(user)

	portLine, err := readLine("Port [22]: ")
	if err != nil {
		return sshconf.Host{}, fmt.Errorf("read port: %w", err)
	}
	if p := strings.TrimSpace(portLine); p != "" {
		port, err := strconv.Atoi(p)
		if err != nil {
			return sshconf.Host{}, fmt.Errorf("invalid port %q: %w", p, err)
		}
		host.Port = port
	}
	return host, nil
}

// confirmHostKey implements auth.ConfirmFunc: it shows the fingerprint of
// an unknown host key and asks whether to trust it.
func confirmHostKey(hostname, fingerprint string) bool {
	prompt := fmt.Sprintf("The authenticity of host %q can't be established.\nKey fingerprint: %s\nTrust this host? [y/N] ", hostname, fingerprint)
	line, _ := readLine(prompt)
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func knownHostsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}

func printSummary(result Result) {
	fmt.Printf("%d succeeded, %d skipped, %d failed\n", len(result.Succeeded), len(result.Skipped), len(result.Failed))
	for path, err := range result.Failed {
		fmt.Fprintf(os.Stderr, "  %s: %v\n", path, err)
	}
}
