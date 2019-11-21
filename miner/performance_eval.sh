if [ $# -lt 2 ]
	then
	echo "Expected ./<script_name> <max threads> <num tests per threadcount>"
	exit 1
fi

echo "Running tests from 1 to ${1} Threads n = ${2} times"
for ((j = 1 ; j <= $1 ; j*=2)); do
	echo "Running tests for ${j} threads"
	for ((i = 0 ; i < $2 ; i++)); do
		go test -run TestCommitTransactionsPerformance -args -threads=$j
	done
done