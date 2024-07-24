#!/usr/bin/env bash

PID="$$"

seed=1234
stages=10
stage_limit=2000
satisfied_after=200

cmd="go run -v . fuzz gen -s $seed -p $stages -l $stage_limit -a $satisfied_after"
fuzz_out="fuzz_out"

mkdir -p "$fuzz_out"

build() {
    file="$1"
    arch_out="$2"

    OUT=$(eval "$cmd $file $arch_out" 2>&1)

    if [ "$?" != 0 ]; then
        echo "ERROR: could not generate fuzzing input."
        echo "$OUT"

        kill -9 "$PID"
        exit 1
    fi
}

for file in ../examples/*.hms; do
    in_out_dir="${file//..\/examples/$fuzz_out}"
    archive_out="${in_out_dir//.hms/.zip}"
    echo "$file" | sed 's#/../examples/#out/#g'

    if [ "$1" = "p" ]; then
        build "$file" "$archive_out" &
    else
        build "$file" "$archive_out"
    fi
done

wait
