import warnings
import sys
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

warnings.simplefilter(action='ignore', category=FutureWarning)

if len(sys.argv) < 4 :
    print("Expected python <script>.py <filename> <path> <graphname>")
    sys.exit()

filename = sys.argv[1]
path = sys.argv[2]
# End the path with / if it is not given
if path[-1] != '/' :
    path += '/'
graphname = sys.argv[3]

# Parse the given csv
df = pd.read_csv(filename,sep="\t", names=['Threads','Throughput'])

# An array of threadcount values
threadVals = df.sort_values(by=['Threads']).Threads.unique()
# The array which will be filled with the average throughput of a given threadcount.
average = []
# The array which will be filled with the std dev throughput of a given threadcount.
stdError = []

# foreach threadcount with data
for index in threadVals :
    #get the throughput values for a given threadcount
    row = df[df["Threads"] == index].Throughput
    #compute the mean
    average.append(row.mean())
    #compute the std dev
    stdError.append(row.std())


# Build the barplot
x_pos = np.arange(len(threadVals))
fig, ax = plt.subplots()
ax.bar(x_pos, average, yerr=stdError, align='center', width=.3 , alpha=0.5, ecolor='black', capsize=10)
ax.set_ylabel('Txns / Second')
ax.set_xlabel('Threads')
ax.set_xticks(x_pos)
ax.set_xticklabels(threadVals)
ax.set_title('Ethereum Transaction Processing')
ax.yaxis.grid(True)
fig.set_tight_layout(True)

# Save the figure
plt.savefig(path + graphname +'.png')