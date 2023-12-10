go build -o a .

for file in ./fuzz/*.hms; do
    echo "Running $file..."
    ./a "$file" both || exit 1
done

wait
