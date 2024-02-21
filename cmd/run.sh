#!/bin/bash
cmd="go run -v . fuzz validate"
# fuzz_out="fuzz_out"

for archive in fuzz_out/*.zip; do
    #in_out_dir="${file//..\/examples/$fuzz_out}"
    #archive_out="${in_out_dir//.hms/.zip}"
    eval "$cmd $archive" || exit 2
done
