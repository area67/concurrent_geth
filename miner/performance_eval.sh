for ((i = 0 ; i <= 1000 ; i++)); do
	go test -run TestCommitTransactionsPerformance
done
