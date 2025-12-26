# s3-cli

A simple CLI to upload an image to an S3-compatible service and print a CDN URL.

## Usage

```bash
s3-cli [options] <image-path> [more-paths...]

Options:
  -config string   Path to config file (default: ~/.s3-cli/config.yaml)
  -debug
                   Enable debug logging
  -init-config
                   Create default config file at config path and exit
  -server string   Server name defined in config
  -unique
                   Use random filename instead of content hash
```

## Config

Default location: `~/.s3-cli/config.yaml`

To create a starter config file:

```bash
s3-cli -init-config -config ~/.s3-cli/config.yaml
```

```yaml
default_server: "primary"

servers:
  primary:
    endpoint: "https://s3.example.com"      # Optional for AWS S3
    region: "us-east-1"
    bucket: "my-bucket"
    access_key_id: "YOUR_ACCESS_KEY"
    secret_access_key: "YOUR_SECRET_KEY"
    session_token: ""                       # Optional
    force_path_style: true                   # For some S3-compatible services
    cdn_base_url: "https://cdn.example.com"
    base_dir: "images"                      # Optional prefix
    path_format: "2006/01/02"               # Go time layout, default is 2006/01/02
    acl: "public-read"                      # Optional

  backup:
    endpoint: "https://s3.backup.example.com"
    region: "us-east-1"
    bucket: "backup-bucket"
    access_key_id: "BACKUP_ACCESS_KEY"
    secret_access_key: "BACKUP_SECRET_KEY"
    cdn_base_url: "https://cdn.backup.example.com"
```

The uploaded object key is:

```
<base_dir>/<path_format>/<random_name>.<ext>
```

If `base_dir` is empty, it is omitted. The filename is replaced with a random hex string, while preserving the original file extension.
