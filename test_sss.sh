bin/vss -setup -project 1 -urls :8000,:8001,:8002
sleep 2
bin/vss -keygen -project 1 -t 1 -urls :8000
sleep 2
result=$(bin/vss -encrypt -project 1 -raw hello -urls :8001)
result2=$(echo $result | jq -r .Msg)
echo $result2
bin/vss -decrypt -project 1 -raw $result2 -urls :8002


#bin/vss -decrypt -project 1 -raw BESj5edJwvHzOwoxtgEZr6xwR-l9Gy1EtpLOptC744zcTwTMQEnE5BTSc7l3Xn67ygKRR0otbPgkjBYIBVXUG7gJVupt8nuiuSSv5CZ6RuQuBxZTbBSZibaUIBbxsUuTWnWMuSz_QWWuA4uhCq8RWV9ddswNhg
