go build -o a .

for file in ./pi_fuzz/*.hms; do
    echo "Running $file..."
    ./a "$file" both &
done

wait
