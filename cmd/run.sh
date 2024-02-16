cmd="go run -v . fuzz validate"
fuzz_out="fuzz_out"

for file in ../examples/*.hms; do
    in_out_dir="${file//..\/examples/$fuzz_out}"
    archive_out="${in_out_dir//.hms/.zip}"
    eval "$cmd $archive_out"
done
