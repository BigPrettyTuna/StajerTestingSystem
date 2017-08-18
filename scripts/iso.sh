#!/bin/bash
dd if=/dev/urandom of={{.FirstPart}}temp.file bs=512 count=2
md5sum {{.FirstPart}}temp.file | awk '{print $1}' > {{.FirstPart}}key.txt
tar -czvf {{.FirstPart}}key.tar.gz {{.FirstPart}}key.txt
genisoimage -o {{.FirstPart}}image.iso {{.FirstPart}}key.tar.gz
