import warnings
import sys
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

warnings.simplefilter(action='ignore', category=FutureWarning)

if len(sys.argv) < 5 :
    print("Expected python <script>.py> <data 100 with tool> <data 100 without tool> <data 200 with tool> <data 200 without tool>  <desired path for graph> <graphname>")
    sys.exit()

filename100Tool = sys.argv[1]
filename100NoTool = sys.argv[2]
filename200Tool = sys.argv[3]
filename200NoTool = sys.argv[4]

path = sys.argv[5]
# End the path with / if it is not given
if path[-1] != '/' :
    path += '/'
graphname = sys.argv[6]

# Parse the given csv
toolDf100 = pd.read_csv(filename100Tool, sep="\t", names=['Threads', 'Throughput'])
noToolDf100 = pd.read_csv(filename100NoTool,sep="\t", names=['Threads','Throughput'])
toolDf200 = pd.read_csv(filename200Tool, sep="\t", names=['Threads', 'Throughput'])
noToolDf200 = pd.read_csv(filename200NoTool,sep="\t", names=['Threads','Throughput'])


# An array of threadcount values
threadVals = toolDf100.sort_values(by=['Threads']).Threads.unique()
# The array which will be filled with the average throughput of a given threadcount.
tool100Average = []
noTool100Average = []
tool200Average = []
noTool200Average = []
# The array which will be filled with the std dev throughput of a given threadcount.
performanceDifference = []

# foreach threadcount with data
for index in threadVals :
    #get the throughput values for a given threadcount
    tool100Average.append(toolDf100[toolDf100["Threads"] == index].Throughput.mean())
    noTool100Average.append(noToolDf100[noToolDf100["Threads"] == index].Throughput.mean())
    tool200Average.append(toolDf200[toolDf200["Threads"] == index].Throughput.mean())
    noTool200Average.append(noToolDf200[noToolDf200["Threads"] == index].Throughput.mean())

efficiency100 = []
efficiency200 = []

for index in range(len(threadVals)) :
    efficiency100.append(tool100Average[index] / noTool100Average[index] * 100)
    efficiency200.append(tool200Average[index] / noTool200Average[index] * 100)

x_pos = np.arange(len(threadVals))

fig, ax = plt.subplots()
efficiency100Line, = ax.plot(x_pos, efficiency100,linewidth=2.5, color='#696969', linestyle='solid')
efficiency200Line, = ax.plot(x_pos, efficiency200,linewidth=2.5, color='#696969', linestyle='dashed')

#ax.fill_between(x_pos, 50, performanceDifference, color='#b8b8b8')
ax.set_ylim([70,100])
ax.set_ylabel('Percent Efficiency')
ax.set_xlabel('Threads')
ax.set_xticks(x_pos)
ax.set_xticklabels(threadVals)
ax.set_title('Correctness Tool Efficiency')
ax.yaxis.grid(True)
ax.legend((efficiency100Line,efficiency200Line),('Txn Pool 100','Txn Pool 200'), loc='center right')
fig.set_tight_layout(True)
# Save the figure
plt.savefig(path + graphname + 'Difference.png')