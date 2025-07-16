# Installation

This section describes how to install `datumctl`.

## Homebrew (macOS)

If you are using macOS and have [Homebrew](https://brew.sh/) installed, you can
install `datumctl` via our official tap:

```bash
# Tap the Datum Cloud formula repository (only needs to be done once)
brew tap datum-cloud/homebrew-tap

# Install datumctl
brew install datumctl

# Upgrade datumctl
brew upgrade datumctl
```

## Pre-built binaries (recommended)

The easiest way to install `datumctl` is by downloading the pre-built binary
for your operating system and architecture from the
[GitHub Releases page](https://github.com/datum-cloud/datumctl/releases).

**Manual Download:**

1.  Go to the [Latest Release](https://github.com/datum-cloud/datumctl/releases/latest).
2.  Find the appropriate archive (`.tar.gz` or `.zip`) for your system (e.g.,
    `datumctl_Linux_x86_64.tar.gz`, `datumctl_Windows_amd64.zip`,
    `datumctl_Darwin_arm64.tar.gz`).
3.  Download and extract the archive.
4.  Move the `datumctl` (or `datumctl.exe`) binary to a directory in your
    system's `PATH` (e.g., `/usr/local/bin` on Linux/macOS, or a custom
    directory you've added to the PATH on Windows).
5.  Ensure the binary is executable (`chmod +x /path/to/datumctl` on
    Linux/macOS).

**Using `curl` (Example for Linux/macOS):**

This example shows how to download and install a specific version. You **must**:

1.  Replace `<version>` with the desired release tag (e.g., `v0.1.0`).
2.  Replace `<archive_filename>` with the exact filename for your OS and
    architecture found on the releases page (e.g.,
    `datumctl_Darwin_arm64.tar.gz`).

```bash
VERSION="<version>"
ARCHIVE="<archive_filename>"
curl -sSL "https://github.com/datum-cloud/datumctl/releases/download/${VERSION}/${ARCHIVE}" | tar xz
sudo mv datumctl /usr/local/bin/
```

> [!NOTE]
> The `sudo mv` command might require administrator privileges. Adjust the
> destination path `/usr/local/bin/` if needed for your system.

## Building from source

If you prefer, you can build `datumctl` from source:

1.  **Prerequisites:**
    *   Go (version 1.21 or later)
    *   Git
2.  **Clone the repository:**
    ```bash
    git clone https://github.com/datum-cloud/datumctl.git
    cd datumctl
    ```
3.  **Build the binary:**
    ```bash
    go build -o datumctl .
    ```
4.  **Install:** Move the resulting `datumctl` binary to a directory in your
    `PATH` as described in the pre-built binaries section.

## Verification

To verify the installation, run:

```bash
datumctl version
```

This should output the installed version of `datumctl`.
