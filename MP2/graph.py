import matplotlib.pyplot as plt
import numpy as np
import time

numNodes = 20
case = 1
baseDir = "eval_logs/"

def bandwidth_graph():
    # for the bandwidth, you should track the average bandwidth across each second of the experiment.
    nodeNums = np.arange(numNodes) + 1
    avgBandwidth = []

    for nodeNum in nodeNums:
        seconds = 0
        #avgBandwidth = []
        with open(baseDir + "bandwidth_node" + str(nodeNum) + ".txt") as file:
            for line in file:
                if line == "\n":
                    line = "0 \n"# continue
                line = line[:-1] # stripping the "\n"
                numbers = line.split(" ")[:-1]
                for i in range(len(numbers)):
                    if numbers[i] != "":
                        numbers[i] = np.float64(numbers[i])
                if numbers != []:
                    bandwidth = np.array(numbers)

                # calculate average bandwidth = length of strings/second1
                if len(avgBandwidth) > seconds:
                    avgBandwidth[seconds] += np.sum(bandwidth)
                else:
                    avgBandwidth.append(np.sum(bandwidth))

                seconds += 1

    # Linear scale plot
    plt.plot(np.arange(len(avgBandwidth)), avgBandwidth, 'm.:', label='bandwidth')
    # plt.scatter((np.arange(seconds/window)*window).tolist(), avgBandwidth, marker='.', c='m', label='bandwidth')
    plt.ylabel('Number of Bytes Downloaded')
    plt.xlabel('Time in Seconds')
    plt.title('Bandwidth Graph Linear Case' + str(case))
    plt.grid()
    plt.savefig(fname=baseDir + 'bandwidth_linear_case' + str(case) + '.png', dpi=200)
    plt.close()

    # Logarithmic scale on the y-axis
    plt.plot(np.arange(len(avgBandwidth)), avgBandwidth, 'bx:', label='bandwidth')
    # plt.scatter((np.arange(seconds/window)*window).tolist(), avgBandwidth, marker='x', c='b', label='bandwidth')
    plt.yscale("log")
    plt.ylabel('Number of Bytes Downloaded')
    plt.xlabel('Time in Seconds')
    plt.title('Bandwidth Graph Log-Scaled Case' + str(case) + '_node' + str(nodeNum))
    plt.grid()
    plt.savefig(fname=baseDir + 'bandwidth_logscale_case' + str(case) + '.png', dpi=200)
    plt.close()
    return

def reachability_graph():
    nodeNums = np.arange(numNodes) + 1

    transactionReach = dict() # format transactionID:number of files seen in
    transactionTime = dict()

    for nodeNum in nodeNums:
        seconds = 0
        avgBandwidth = []
        window = 1 # 1 seconds
        with open(baseDir + "transactions_node" + str(nodeNum) + ".txt") as file:
            for line in file:
                line = line[:-1] # stripping the "\n"
                line = line.strip()
                currID = line.split(" ")[1]
                tTransactiontime = line.split(" ")[2]
                if currID not in transactionReach.keys():
                    transactionReach[currID] = 0
                    transactionTime[currID] = tTransactiontime
                transactionReach[currID] += 1

    # sort by transaction time
    xSorted = sorted(transactionTime.items(), key=lambda x: x[1])
    y = []
    for x in xSorted:
        key = x[0]
        y.append(transactionReach[key])
    x = np.arange(len(y))
    xticks = [item[0] for item in xSorted]

    #"""
    # Bar Graph
    plt.bar(x, y, label='Reachability') #, 'm.:', )
    #plt.xticks(x, xticks)
    plt.ylabel('Number of Nodes reached')
    plt.xlabel('Time Transaction Created At')
    plt.title('Reachability Graph Case' + str(case))
    plt.grid()
    #plt.show()
    plt.savefig(fname = baseDir + 'reachability_case' + str(case) + '.png', dpi=500)
    plt.close()
    return

def propagation_delay_graph():
    transactionReach = dict() # format transactionID:[time transaction reached a node]
    transactionTime = dict() # format transactionID:transactionTime

    nodeNums = np.arange(numNodes) + 1
    for nodeNum in nodeNums:
        with open(baseDir + "transactions_node" + str(nodeNum) + ".txt") as file:
            for line in file:
                line = line[:-1] # stripping the "\n"
                line = line.strip()
                loggingTime, currID, tTransactiontime = line.split(" ")
                loggingTime=np.float64(loggingTime)
                tTransactiontime=np.float64(tTransactiontime)

                if currID not in transactionReach.keys():
                    #transactionReach[currID] = []
                    transactionReach[currID] = [loggingTime-tTransactiontime] #np.array([loggingTime])
                    transactionTime[currID] = tTransactiontime
                else:
                    transactionReach[currID].append(loggingTime-tTransactiontime)
                    #np.append(transactionReach[currID], [loggingTime])

    # sort by transaction time
    minList = [] # min propagation delay
    maxList = []
    medianList = []

    sortedKeys = sorted(transactionTime.items(), key=lambda x: x[1])

    for item in sortedKeys:
        key = item[0]
        data = np.array(transactionReach[key])
        if len(data) > 1:
            sortedData = np.sort(data)
            minList.append(sortedData[1])
        else:
            minList.append(data[0])
        maxList.append(np.max(data))
        medianList.append(np.median(data))

    x = np.arange(len(sortedKeys))
    xticks = [item[0] for item in sortedKeys]

    fig = plt.figure()
    ax1 = fig.add_subplot(111)
    ax1.plot(x, minList, 'C1x-', label='minimum delay')
    ax1.plot(x, maxList, 'C3x-', label='maximum')
    ax1.plot(x, medianList, 'C2.-', label='median')
    plt.legend(loc='upper right');
    ax1.set_ylabel('Delay in Seconds')
    ax1.set_xlabel('Time in Seconds')
    ax1.set_title('Propagation Delay Graph Case' + str(case))
    plt.savefig(fname = baseDir + 'propagation_delay_case' + str(case) + '.png', dpi=200)
    plt.close()

    return

propagation_delay_graph()
reachability_graph()
bandwidth_graph()
