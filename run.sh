for file in ./fuzz/*.hms; do
    echo "Running $file..."
    ./a "$file" 0 || exit 1
done

wait
