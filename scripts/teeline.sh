#!/bin/bash

get_bookings () {
  local date_offset=$1

  # get date offset in the proper format, i.e. "10-10-2023"
  NEXT_DATE=$(date -v +${date_offset}d +%m-%d-%Y)
  
  curl -s "https://www.sevenrooms.com/api-yoa/availability/widget/range?venue=teeline&time_slot=17:00&party_size=4&halo_size_interval=24&start_date=${NEXT_DATE}&num_days=1&channel=SEVENROOMS_WIDGET&selected_lang_code=en" | jq -r '.data.availability | keys[] as $k | .[$k] | .[0].times[] | select(.type | contains("book")) | [(now | strflocaltime("%Y-%m-%d %H:%M:%S")), $k, .time] | @csv' >> "${NEXT_DATE}.csv"
# "2023-10-10T18:10:39-0600"
}

# START_DATE="10-10-2023"
# START_DATE=$(date +%m-%d-%Y)
# date -v +0d +%m-%d-%Y
START_DATE=$(date -v +1d)
NUM_DAYS=1


for i in {1..2}
do
  get_bookings $i
done

# printf -v date '%(%d-%m-%YY)T\n' -1


# echo $START_DATE

# curl -s "https://www.sevenrooms.com/api-yoa/availability/widget/range?venue=teeline&time_slot=17:00&party_size=4&halo_size_interval=24&start_date=${START_DATE}&num_days=${NUM_DAYS}&channel=SEVENROOMS_WIDGET&selected_lang_code=en" | jq -r '.data.availability | keys[] as $k | .[$k] | .[0].times[] | select(.type | contains("book")) | [$k, .time] | @csv'

