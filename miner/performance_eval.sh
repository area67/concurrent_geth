if [ $# -lt 3 ]
	then
	echo "Expected ./<script_name> <max threads> <num tests per threadcount> <output file name>"
	exit 1
fi

echo "Running tests from 1 to ${1} Threads n = ${2} times"
for ((j = 1 ; j <= $1 ; j*=2)); do
	echo "Running tests for ${j} threads"
	for ((i = 0 ; i < $2 ; i++)); do
		go test -run TestCommitTransactionsPerformance -args -threads=$j -output="$3"

	done
done