# EC2 AutoRestore

Easily backup and restore a set of EC2 instances based on a shared tag.

## Building
With go installed, simply call:
```
$ go build
```

## Basic Usage
### Backup
Configure your environment to authenticate to your AWS accounts as normal.
Label your EC2 instances with a tag of "backup" and a meaningful key.
Think of a name for your backup.
Finally, run:
```
$ ec2-autorestore backup [the key] [your backup name]
```
### Restore
To restore, run:
```
$ ec2-autorestore restore [your backup name]
```
### Help
For further usage instructions, simply call ec2-autorestore with help
```
$ ec2-autorestore help
```