package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type Disk struct {
	DeviceIdentifier string `json:"DeviceIdentifier"`
	Content          string `json:"Content"`
	VolumeName       string `json:"VolumeName"`
	Size             int64  `json:"Size"`
	APFSVolumes      []Disk `json:"APFSVolumes"`
	Partitions       []Disk `json:"Partitions"`
}

func hrSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	s := float64(size)
	i := 0
	for s >= 1024 && i < len(units)-1 {
		s /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", s, units[i])
}

func listDisks() error {
	cmd := exec.Command("diskutil", "list", "-plist")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	pl := exec.Command("plutil", "-convert", "json", "-o", "-", "--", "-")
	pl.Stdin = bytes.NewReader(out)
	jsonOut, err := pl.Output()
	if err != nil {
		return err
	}

	var root map[string]interface{}
	if err := json.Unmarshal(jsonOut, &root); err != nil {
		return err
	}

	all, ok := root["AllDisksAndPartitions"].([]interface{})
	if !ok {
		return fmt.Errorf("unexpected plist structure")
	}

	for i, d := range all {
		disk := parseDisk(d.(map[string]interface{}))
		printDiskTree(disk, "", true)

		// spacing between parent disks
		if i < len(all)-1 {
			fmt.Println()
		}
	}

	return nil
}

func parseDisk(m map[string]interface{}) Disk {
	d := Disk{}
	if v, ok := m["DeviceIdentifier"].(string); ok {
		d.DeviceIdentifier = v
	}
	if v, ok := m["Content"].(string); ok {
		d.Content = v
	}
	if v, ok := m["VolumeName"].(string); ok {
		d.VolumeName = v
	}
	if v, ok := m["Size"].(float64); ok {
		d.Size = int64(v)
	}

	if parts, ok := m["Partitions"].([]interface{}); ok {
		for _, p := range parts {
			d.Partitions = append(d.Partitions, parseDisk(p.(map[string]interface{})))
		}
	}
	if vols, ok := m["APFSVolumes"].([]interface{}); ok {
		for _, v := range vols {
			d.APFSVolumes = append(d.APFSVolumes, parseDisk(v.(map[string]interface{})))
		}
	}
	return d
}

func printDiskTree(d Disk, prefix string, isLast bool) {
	branch := "├─ "
	if isLast {
		branch = "└─ "
	}

	label := "-"
	if d.VolumeName != "" {
		label = d.VolumeName
	}

	fs := "-"
	if d.Content != "" {
		fs = d.Content
	} else if len(d.APFSVolumes) > 0 {
		fs = "APFS"
	}

	size := "-"
	if d.Size > 0 {
		size = hrSize(d.Size)
	}

	line := fmt.Sprintf("%s%s/dev/%s [%s]", prefix, branch, d.DeviceIdentifier, fs)
	if label != "-" {
		line += " " + label
	}
	if size != "-" {
		line += " (" + size + ")"
	}
	fmt.Println(line)

	children := append([]Disk{}, d.Partitions...)
	children = append(children, d.APFSVolumes...)

	for i, c := range children {
		last := i == len(children)-1
		newPrefix := prefix
		if isLast {
			newPrefix += "   "
		} else {
			newPrefix += "│  "
		}
		printDiskTree(c, newPrefix, last)
	}
}

func getDiskSize(path string) (int64, error) {
	cmd := exec.Command("diskutil", "info", "-plist", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	pl := exec.Command("plutil", "-convert", "json", "-o", "-", "--", "-")
	pl.Stdin = bytes.NewReader(out)
	jsonOut, err := pl.Output()
	if err != nil {
		return 0, err
	}

	var info map[string]interface{}
	if err := json.Unmarshal(jsonOut, &info); err != nil {
		return 0, err
	}

	if v, ok := info["TotalSize"].(float64); ok {
		return int64(v), nil
	}
	return 0, fmt.Errorf("disk size not found")
}

func cloneDisk(src, dst string) error {
	totalSize, err := getDiskSize(src)
	if err != nil {
		return fmt.Errorf("failed to get disk size: %v", err)
	}

	cmdStr := fmt.Sprintf("dd if=%s bs=1m | pv -s %d | dd of=%s bs=1m", src, totalSize, dst)
	fmt.Println("Running (sudo required):", cmdStr)

	cmd := exec.Command("sudo", "bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func writeDisk(src, dst string) error {
	totalSize, err := getDiskSize(dst)
	if err != nil {
		return fmt.Errorf("failed to get disk size: %v", err)
	}

	fmt.Printf("⚠ WARNING: This will overwrite all data on %s\n", dst)
	fmt.Print("Type YES to continue: ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "YES" {
		fmt.Println("Aborted.")
		return nil
	}

	cmdStr := fmt.Sprintf("pv -s %d %s | dd of=%s bs=1m", totalSize, src, dst)
	fmt.Println("Running (sudo required):", cmdStr)

	cmd := exec.Command("sudo", "bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: osximg {list|clone|write}")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		if err := listDisks(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	case "clone":
		if len(os.Args) != 4 {
			fmt.Println("Usage: osximg clone /dev/diskX /path/to/file.img")
			os.Exit(1)
		}
		if err := cloneDisk(os.Args[2], os.Args[3]); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	case "write":
		if len(os.Args) != 4 {
			fmt.Println("Usage: osximg write /path/to/file.img /dev/diskX")
			os.Exit(1)
		}
		if err := writeDisk(os.Args[2], os.Args[3]); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Usage: osximg {list|clone|write}")
		os.Exit(1)
	}
}
