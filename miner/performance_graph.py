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

if path[-1] != '/' :
    path += '/'

graphname = sys.argv[3]
df = pd.read_csv(filename,sep="\t")

threadVals = df.sort_values(by=['Threads']).Threads.unique()
average = []
stdError = []
for index in threadVals :
    row = df[df["Threads"] == index].Throughput
    average.append(row.mean())
    stdError.append(row.std())

x_pos = np.arange(len(threadVals))

fig, ax = plt.subplots()
ax.bar(x_pos, average, yerr=stdError, align='center', width=.3 , alpha=0.5, ecolor='black', capsize=10)
ax.set_ylabel('Txns / Second')
ax.set_xlabel('Threads')
ax.set_xticks(x_pos)
ax.set_xticklabels(threadVals)
ax.set_title('Ethereum Transaction Processing')
ax.yaxis.grid(True)

# Save the figure and show
fig.set_tight_layout(True)

plt.savefig(path + graphname +'.png')


#ax = averages.plot.bar(x='Threads', y='Txs/Sec', rot=0)
#fig = ax.get_figure()
#fig.savefig('/Users/admin/Desktop/test.png')