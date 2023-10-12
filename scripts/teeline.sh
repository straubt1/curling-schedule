#!/bin/bash

get_next_date () {
  local date_offset=$1  

  case `uname` in

  Darwin)
    echo "Darwin"
    NEXT_DATE=$(date -v +${date_offset}d +%m-%d-%Y)
    # TIMESTAMP=`date -v -${EXPIRY_DAYS}d +%Y-%m-%d`
    ;;
  Linux)
    echo "Linux"
    NEXT_DATE=$(date -u --date="+${date_offset} day" +%m-%d-%Y)
    # TIMESTAMP=`date -u --date="-${EXPIRY_DAYS} day" +%Y-%m-%d`
    ;;
  *)
    echo "Platform not supported. Exiting.";
    exit 1;
    ;;
esac

}
get_bookings () {
  local date_offset=$1

  # get date offset in the proper format, i.e. "10-10-2023"
  NEXT_DATE=$(get_next_date ${date_offset})
  
  curl -s "https://www.sevenrooms.com/api-yoa/availability/widget/range?venue=teeline&time_slot=17:00&party_size=4&halo_size_interval=24&start_date=${NEXT_DATE}&num_days=1&channel=SEVENROOMS_WIDGET&selected_lang_code=en" | jq -r '.data.availability | keys[] as $k | .[$k] | .[0].times[] | select(.type | contains("book")) | [(now | strflocaltime("%Y-%m-%d %H:%M:%S")), $k, .time] | @csv' >> "${NEXT_DATE}.csv"
}

for i in {0..6}
do
  get_bookings $i
done

# printf -v date '%(%d-%m-%YY)T\n' -1


# echo $START_DATE

# curl -s "https://www.sevenrooms.com/api-yoa/availability/widget/range?venue=teeline&time_slot=17:00&party_size=4&halo_size_interval=24&start_date=${START_DATE}&num_days=${NUM_DAYS}&channel=SEVENROOMS_WIDGET&selected_lang_code=en" | jq -r '.data.availability | keys[] as $k | .[$k] | .[0].times[] | select(.type | contains("book")) | [$k, .time] | @csv'

