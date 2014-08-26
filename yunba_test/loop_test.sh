MAX=100
 
for (( i = 0; i < MAX ; i ++ ))
do
    echo "#################################" $i "####################################"
  
    go test -v 
    if [ $? -ne 0 ]
    then
        echo "test failed! No:" $i
        break
    fi
    # 或者这样
    #  echo $[$i * $i]
done
