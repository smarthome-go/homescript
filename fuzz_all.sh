./fuzz.sh ./examples/primes.hms prime_fuzz &
./fuzz.sh ./examples/fizzbuzz.hms fizz_fuzz &
./fuzz.sh ./examples/box.hms box_fuzz &
./fuzz.sh ./examples/binary.hms binary_fuzz &
./fuzz.sh ./examples/dev.hms dev_fuzz &
./fuzz.sh ./examples/pow.hms pow_fuzz &
./fuzz.sh ./examples/pi.hms pi_fuzz &
./fuzz.sh ./examples/e.hms e_fuzz &
./fuzz.sh ./examples/apery.hms apery_fuzz &
./fuzz.sh ./examples/matrix.hms matrix_fuzz &
wait
