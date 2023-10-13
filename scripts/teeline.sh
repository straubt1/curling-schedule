#!/bin/bash

get_next_date () {
  local date_offset=$1  

  case `uname` in

  Darwin)
    NEXT_DATE=$(TZ=":US/Central" date -v +${date_offset}d +%m-%d-%Y)
    # TIMESTAMP=`date -v -${EXPIRY_DAYS}d +%Y-%m-%d`
    ;;
  Linux)
    NEXT_DATE=$(TZ=":US/Central" date -u --date="+${date_offset} day" +%m-%d-%Y)
    # TIMESTAMP=`date -u --date="-${EXPIRY_DAYS} day" +%Y-%m-%d`
    ;;
  *)
    echo "Platform not supported. Exiting.";
    exit 1;
    ;;
esac
echo $NEXT_DATE
}

create_date_file () {
  local NEXT_FILENAME=$1

  if ! test -f ${NEXT_FILENAME}; then
    echo "File ${NEXT_FILENAME} does not exist, creating it now with CSV headers"
    echo "Query Date,Available Times" > ${NEXT_FILENAME}
  fi
}

get_bookings () {
  local date_offset=$1

  NEXT_DATE=$(get_next_date ${date_offset}) # get date offset in the proper format, i.e. "10-10-2023"
  NEXT_FILENAME="dates/${NEXT_DATE}.csv"
  QUERY_DATE=$(TZ=":US/Central" date +"%m-%d-%Y %I:%M:%S %p")
  
  create_date_file ${NEXT_FILENAME}
  # Get times that are open, returns a string with those times for a single day
  TIMES=`curl -s "https://www.sevenrooms.com/api-yoa/availability/widget/range?venue=teeline&time_slot=17:00&party_size=4&halo_size_interval=24&start_date=${NEXT_DATE}&num_days=1&channel=SEVENROOMS_WIDGET&selected_lang_code=en" | jq -r '.data.availability | keys[] as $k | .[$k] | .[0].times[] | select(.type == "book").time'`
  
  # Replace new lines with "; " since jq doesn't like to do what I ask...
  TIMES=${TIMES//$'\n'/; }
  echo "${QUERY_DATE},${TIMES}" >> ${NEXT_FILENAME}
}

# get_bookings 0

# cat times.json| jq -r '.data.availability | keys[] as $k | .[$k] | .[0].times[] | select(.type == "book")'
# TIMES=`cat times.json | jq -r '.data.availability | keys[] as $k | .[$k] | .[0].times[] | select(.type == "book").time'`
# echo $TIMES
# cat times.json| jq -r '.data.availability | keys[] as $k | .[$k] | .[0].times[] | select(.type | contains("book")) | [(now | strflocaltime("%Y-%m-%d %H:%M:%S")), $k, .time] | join(",")'

# get_bookings 0 
for i in {0..6}
do
  get_bookings $i
done

# printf -v date '%(%d-%m-%YY)T\n' -1


# echo $START_DATE

# curl -s "https://www.sevenrooms.com/api-yoa/availability/widget/range?venue=teeline&time_slot=17:00&party_size=4&halo_size_interval=24&start_date=${START_DATE}&num_days=${NUM_DAYS}&channel=SEVENROOMS_WIDGET&selected_lang_code=en" | jq -r '.data.availability | keys[] as $k | .[$k] | .[0].times[] | select(.type | contains("book")) | [$k, .time] | @csv'

