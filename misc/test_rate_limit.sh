#!/bin/bash

# Start measuring time
start=$(date +%s.%N)

num_calls=10
# shellcheck disable=SC2034
for i in $(seq 1 $num_calls);
do
  curl -s -o /dev/null -w "%{http_code}" --location 'https://neal.fun/api/infinite-craft/pair?first=Acid%20Rain&second=Acid%20Rain' --compressed -H 'Referer: https://neal.fun/infinite-craft/' -H 'User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0'
  echo
done

# End measuring time
end=$(date +%s.%N)

# Calculate and print the elapsed time
runtime=$(echo "$end - $start" | bc)
echo "Infinite Craft endpoint hit $num_calls times in the past $runtime seconds"
