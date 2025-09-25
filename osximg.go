package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const version = "v0.1.4"

type Disk struct {
	DeviceIdentifier string `json:"DeviceIdentifier"`
	Content          string `json:"Content"`
	VolumeName       string `json:"VolumeName"`
	Size             int64  `json:"Size"`
	APFSVolumes      []Disk `json:"APFSVolumes"`
	Partitions       []Disk `json:"Partitions"`
}

func usage() string {
	return fmt.Sprintf("osximg version %s\n\nUsage: osximg {list|clone|write}", version)
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

	// Combine label and fs, label comes first
	labelFs := label
	if fs != "-" {
		if label != "-" {
			labelFs = fmt.Sprintf("%s | %s", label, fs)
		} else {
			labelFs = fs
		}
	}

	size := "-"
	if d.Size > 0 {
		size = hrSize(d.Size)
	}

	line := fmt.Sprintf("%s%s/dev/%s [%s]", prefix, branch, d.DeviceIdentifier, labelFs)
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
	// Get source image size
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source image size: %v", err)
	}
	imageSize := info.Size()

	// Check if target disk is internal
	internal, err := isInternalDisk(dst)
	if err == nil && internal {
		fmt.Printf("⚠ WARNING: %s is an INTERNAL disk!\n", dst)
		fmt.Print("Are you absolutely sure you want to continue? Type INTERNAL to confirm: ")
		var confirmInternal string
		fmt.Scanln(&confirmInternal)
		if confirmInternal != "INTERNAL" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Printf("⚠ WARNING: This will overwrite all data on %s\n", dst)
	fmt.Printf("Source image size: %s (%d bytes)\n", hrSize(imageSize), imageSize)
	fmt.Print("Type YES to continue: ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "YES" {
		fmt.Println("Aborted.")
		return nil
	}

	fmt.Printf("Writing %s → %s\n", src, dst)

	cmdStr := fmt.Sprintf("pv -s %d %s | dd of=%s bs=1m", imageSize, src, dst)
	fmt.Println("Running (sudo required):", cmdStr)

	cmd := exec.Command("sudo", "bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func isInternalDisk(path string) (bool, error) {
	cmd := exec.Command("diskutil", "info", "-plist", path)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}

	pl := exec.Command("plutil", "-convert", "json", "-o", "-", "--", "-")
	pl.Stdin = bytes.NewReader(out)
	jsonOut, err := pl.Output()
	if err != nil {
		return false, err
	}

	var info map[string]interface{}
	if err := json.Unmarshal(jsonOut, &info); err != nil {
		return false, err
	}

	if v, ok := info["Internal"].(bool); ok {
		return v, nil
	}
	return false, nil
}

// confirmRdisk checks if user passed /dev/diskX and prompts to use /dev/rdiskX
func confirmRdisk(disk string) string {
	re := regexp.MustCompile(`^/dev/disk(\d+)$`)
	matches := re.FindStringSubmatch(disk)
	if matches != nil {
		rdisk := "/dev/rdisk" + matches[1]
		fmt.Printf("%s detected. Do you want to use %s instead for faster performance? [y/N]: ", disk, rdisk)

		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) == "y" {
			return rdisk
		}
	}
	return disk
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage())
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
		disk := confirmRdisk(os.Args[2])
		imgPath := os.Args[3]
		cloneDisk(disk, imgPath)

	case "write":
		if len(os.Args) != 4 {
			fmt.Println("Usage: osximg write /path/to/file.img /dev/diskX")
			os.Exit(1)
		}
		imgPath := os.Args[2]
		disk := confirmRdisk(os.Args[3])
		writeDisk(imgPath, disk)

	case "version":
		fmt.Println(version)

	default:
		fmt.Println(usage())
		os.Exit(1)
	}
}
