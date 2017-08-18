#!/bin/bash
dd if=/dev/urandom of={{.FirstPart}}md5.file bs=512 count=2
md5sum {{.FirstPart}}md5.file | awk '{print $1}' > {{.FirstPart}}temp.txt