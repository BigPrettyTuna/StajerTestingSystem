#!/bin/bash
rightip=$(( ( RANDOM %8) + 1 ))
rightkey=$(( ( RANDOM %8) + 1 ))
rightdb=$(( ( RANDOM %8) + 1 ))
for (( j=1; j <=4; j++ ))
do
for (( i=1; i <= $(( ( RANDOM % 20 ) + ( RANDOM % 20 )  + ( RANDOM % 20 ) + 43 )); i++ ))
do
key1="$(cat /dev/urandom | tr -dc 'A-Z' | fold -w 5 | head -n 1)"
key2="$(cat /dev/urandom | tr -dc 'A-Za-z' | fold -w 5 | head -n 1)"
key3="$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 5 | head -n 1)"
key4="$(cat /dev/urandom | tr -dc 'a-z' | fold -w 5 | head -n 1)"
key5="$(cat /dev/urandom | tr -dc 'A-Z' | fold -w 5 | head -n 1)"
ip1="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 2 )) | head -n 1)"
ip2="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 2 )) | head -n 1)"
ip3="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 1 )) | head -n 1)"
ip4="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 2 )) | head -n 1)"
db1="$(echo "<DBName>""$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w $(( ( RANDOM % 8 ) + ( RANDOM % 7 )  + ( RANDOM % 6 ) + 4 )) | head -n 1)""</DBName>")"
#echo $str1" "$str2 >> {{.FirstPart}}file.txt
int1=$(( ( RANDOM % 50 )  + 20 ))
int2=$(( ( RANDOM % 50 )  + 20 ))
int3=$(( ( RANDOM % 50 )  + 20 ))
int4=$(( ( RANDOM % 50 )  + 20 ))
if [ $int1 -gt $int2 ]
then
if [ $int3 -gt $int4 ]
then
echo $key1"-"$key2"-"$key3"-"$key4"-"$key5" "$ip1"."$ip2"."$ip3"."$ip4" "$db1 >> {{.FirstPart}}file.txt
else
echo $ip1"."$ip2"."$ip3"."$ip4" "$key1"-"$key2"-"$key3"-"$key4"-"$key5" "$db1 >> {{.FirstPart}}file.txt
fi
else
echo $db1" "$key1"-"$key2"-"$key3"-"$key4"-"$key5" "$ip1"."$ip2"."$ip3"."$ip4 >> {{.FirstPart}}file.txt
fi
done
if [ $j -eq $rightip ]
then
key1="$(cat /dev/urandom | tr -dc 'A-Z' | fold -w 5 | head -n 1)"
key2="$(cat /dev/urandom | tr -dc 'A-Za-z' | fold -w 5 | head -n 1)"
key3="$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 5 | head -n 1)"
key4="$(cat /dev/urandom | tr -dc 'a-z' | fold -w 5 | head -n 1)"
key5="$(cat /dev/urandom | tr -dc 'A-Z' | fold -w 5 | head -n 1)"
db1="$(echo "<DBName>""$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w $(( ( RANDOM % 8 ) + ( RANDOM % 7 )  + ( RANDOM % 6 ) + 4 )) | head -n 1)""</DBName>")"
echo $db1" "$key1"-"$key2"-"$key3"-"$key4"-"$key5" ""212.193.32.134" >> {{.FirstPart}}file.txt
fi
if [ $j -eq $rightkey ]
then
ip1="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 2 )) | head -n 1)"
ip2="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 2 )) | head -n 1)"
ip3="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 1 )) | head -n 1)"
ip4="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 2 )) | head -n 1)"
db1="$(echo "<DBName>""$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w $(( ( RANDOM % 8 ) + ( RANDOM % 7 )  + ( RANDOM % 6 ) + 4 )) | head -n 1)""</DBName>")"
echo $db1" ""AVWEF-EFfQE-wD3FF-asFew-WEFWQ"" "$ip1"."$ip2"."$ip3"."$ip4 >> {{.FirstPart}}file.txt
fi
if [ $j -eq $rightdb ]
then
key1="$(cat /dev/urandom | tr -dc 'A-Z' | fold -w 5 | head -n 1)"
key2="$(cat /dev/urandom | tr -dc 'A-Za-z' | fold -w 5 | head -n 1)"
key3="$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 5 | head -n 1)"
key4="$(cat /dev/urandom | tr -dc 'a-z' | fold -w 5 | head -n 1)"
key5="$(cat /dev/urandom | tr -dc 'A-Z' | fold -w 5 | head -n 1)"
ip1="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 2 )) | head -n 1)"
ip2="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 2 )) | head -n 1)"
ip3="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 1 )) | head -n 1)"
ip4="$(cat /dev/urandom | tr -dc '0-9' | fold -w $(( ( RANDOM % 3 ) + 2 )) | head -n 1)"
echo "<DBName>datBa44eOMG1</DBName>"" "$key1"-"$key2"-"$key3"-"$key4"-"$key5" "$ip1"."$ip2"."$ip3"."$ip4 >> {{.FirstPart}}file.txt
fi
echo $j
done
