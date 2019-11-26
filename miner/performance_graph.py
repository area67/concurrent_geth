import warnings
import sys
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

warnings.simplefilter(action='ignore', category=FutureWarning)

if len(sys.argv) < 5 :
    print("Expected python <script>.py> <data with tool> <data without tool> <desired path for graph> <graphname>")
    sys.exit()

filenameTool = sys.argv[1]
filenameNoTool = sys.argv[2]

path = sys.argv[3]
# End the path with / if it is not given
if path[-1] != '/' :
    path += '/'
graphname = sys.argv[4]

# Parse the given csv
toolDf = pd.read_csv(filenameTool, sep="\t", names=['Threads', 'Throughput'])

# An array of threadcount values
threadVals = toolDf.sort_values(by=['Threads']).Threads.unique()
# The array which will be filled with the average throughput of a given threadcount.
toolAverage = []
# The array which will be filled with the std dev throughput of a given threadcount.
toolStdError = []
performanceDifference = []

# foreach threadcount with data
for index in threadVals :
    #get the throughput values for a given threadcount
    row = toolDf[toolDf["Threads"] == index].Throughput
    #compute the mean
    toolAverage.append(row.mean())
    #compute the std dev
    toolStdError.append(row.std())


# Parse the given csv
noToolDf = pd.read_csv(filenameNoTool,sep="\t", names=['Threads','Throughput'])

# An array of threadcount values
threadVals = noToolDf.sort_values(by=['Threads']).Threads.unique()
# The array which will be filled with the average throughput of a given threadcount.
noToolAverage = []
# The array which will be filled with the std dev throughput of a given threadcount.
noToolStdError = []


# foreach threadcount with data
for index in threadVals :
    #get the throughput values for a given threadcount
    row = noToolDf[noToolDf["Threads"] == index].Throughput
    #compute the mean
    noToolAverage.append(row.mean())
    #compute the std dev
    noToolStdError.append(row.std())


for index in range(len(threadVals)) :
    performanceDifference.append(toolAverage[index] / noToolAverage[index] * 100)

# get control data
#ctlData = pd.read_csv("control.txt",sep="\t", names=['Threads','Throughput'])
# An array of threadcount values
#ctlThreadVals = ctlData.sort_values(by=['Threads']).Threads.unique()
# The array which will be filled with the average throughput of a given threadcount.
#ctlAverage = []
# The array which will be filled with the std dev throughput of a given threadcount.
#ctlStdError = []

# foreach threadcount with data
#for index in ctlThreadVals :
    #get the throughput values for a given threadcount
 #   row = ctlData[ctlData["Threads"] == index].Throughput
  #  #compute the mean
   # ctlAverage.append(row.mean())
    #compute the std dev
    #ctlStdError.append(row.std())

# Build the barplot
x_pos = np.arange(len(threadVals))
fig, ax = plt.subplots()
# ctlLine, = ax.plot(x_pos,ctlAverage,color="red",linewidth=2.5, linestyle="--")

dataBars = ax.bar(x_pos-.15, toolAverage, yerr=toolStdError, color='#5c584f', align='center', width=.3 , alpha=0.5, ecolor='black', capsize=10)
dataBars2 = ax.bar(x_pos+.15, noToolAverage, yerr=noToolStdError, color='#d49402', align='center', width=.3 , alpha=0.5, ecolor='black', capsize=10)

ax.set_ylabel('Txns / Second')
ax.set_xlabel('Threads')
ax.set_xticks(x_pos)
ax.set_xticklabels(threadVals)
ax.set_title('Ethereum Transaction Processing')
ax.yaxis.grid(True)
fig.set_tight_layout(True)
ax.legend((dataBars,dataBars2),('With Tool','W/o Tool'))
# Save the figure
plt.savefig(path + graphname +'.png')



fig, ax = plt.subplots()
differenceLine, = ax.plot(x_pos, performanceDifference,linewidth=2.5, color='#696969')
ax.fill_between(x_pos, 50, performanceDifference, color='#b8b8b8')
ax.set_ylim([50,100])
ax.set_ylabel('Percent Efficiency')
ax.set_xlabel('Threads')
ax.set_xticks(x_pos)
ax.set_xticklabels(threadVals)
ax.set_title('Correctness Tool Efficiency')
ax.yaxis.grid(True)
fig.set_tight_layout(True)
# Save the figure
plt.savefig(path + graphname + 'Difference.png')