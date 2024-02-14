#!/bin/bash

# Start measuring time
start=$(date +%s.%N)

num_calls=100
# shellcheck disable=SC2034
for i in $(seq 1 $num_calls);
do
  curl -s -o /dev/null -w "%{http_code}" --location 'https://neal.fun/api/infinite-craft/pair?first=Acid%20Rain&second=Acid%20Rain' --header 'Referer: https://neal.fun/infinite-craft/'
  echo
done

# End measuring time
end=$(date +%s.%N)

# Calculate and print the elapsed time
runtime=$(echo "$end - $start" | bc)
echo "Infinite Craft endpoint hit $num_calls times in the past $runtime seconds"
