./fuzz.sh ./examples/primes.hms prime_fuzz &
./fuzz.sh ./examples/fizzbuzz.hms fizz_fuzz &
./fuzz.sh ./examples/box.hms box_fuzz &
./fuzz.sh ./examples/binary.hms binary_fuzz &
./fuzz.sh ./examples/dev.hms dev_fuzz &
wait
