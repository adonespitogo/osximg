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
└─ /dev/disk3 [Apple_APFS_Container] (460.4 GB)
   ├─ /dev/disk3s1 [Macintosh HD - Data] (460.4 GB)
   ├─ /dev/disk3s2 [Update] (460.4 GB)
   ├─ /dev/disk3s3 [Macintosh HD] (460.4 GB)
   ├─ /dev/disk3s3s1 [Macintosh HD] (460.4 GB)
   ├─ /dev/disk3s4 [Preboot] (460.4 GB)
   ├─ /dev/disk3s5 [Recovery] (460.4 GB)
   └─ /dev/disk3s6 [VM] (460.4 GB)

└─ /dev/disk9 [GUID_partition_scheme] (14.8 GB)
   └─ /dev/disk9s1 [Linux Filesystem] (14.7 GB)
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

