#!/bin/bash
seed=1234
stages=10
stage_limit=2000
satisfied_after=200

cmd="go run -v . fuzz gen -s $seed -p $stages -l $stage_limit -a $satisfied_after"
fuzz_out="fuzz_out"

mkdir -p "$fuzz_out"

for file in ../examples/*.hms; do
    in_out_dir="${file//..\/examples/$fuzz_out}"
    archive_out="${in_out_dir//.hms/.zip}"
    eval "$cmd $file $archive_out" &
     # echo "$file" | sed 's#/../examples/#out/#g'
done

wait
