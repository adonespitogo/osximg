# osximg

A simple macOS command-line tool to **list disks/volumes** and **clone/write raw `.img` images** using `dd`.

## Features

- 📂 List physical disks and partitions in a tree-style layout.
- 💾 Clone any disk to a raw `.img` file.
- 🔄 Write `.img` files back to a disk.
- 📊 Shows progress (bytes transferred, total size, speed).

---

## Installation

### Using Go

Make sure you have [Go installed](https://golang.org/doc/install) (Go 1.22+).

```bash
go install github.com/adonespitogo/osximg@latest
```

Ensure your Go binary path is in your `PATH`:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

Verify installation:

```bash
osximg --help
```

---

## Usage

### List Disks

```bash
osximg list
```

Example output:

```
/dev/disk0 (500GB, GUID_partition_scheme)
  ├─ /dev/disk0s1  Apple_APFS_ISC      524MB
  ├─ /dev/disk0s2  Apple_APFS          494GB
  └─ /dev/disk0s3  Apple_APFS_Recovery 5.3GB

/dev/disk8 (4TB, GUID_partition_scheme)
  └─ /dev/disk8s1  Microsoft Basic Data 4TB  [4TBSSD]
```

---

### Clone a Disk to `.img`

```bash
sudo osximg clone /dev/disk5 /path/to/file.img
```

---

### Write `.img` to a Disk

⚠️ **Warning**: This will overwrite all data on the target disk!

```bash
sudo osximg write /path/to/file.img /dev/disk5
```

---

## Notes

- Always double-check your target disk (`/dev/diskX`) before cloning or writing.
- `sudo` is required for raw disk access.
- Uses native `dd` under the hood with real-time progress parsing.

---

## License

MIT License © [Adones Pitogo](https://github.com/adonespitogo)

