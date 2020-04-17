import matplotlib.pyplot as plt
import numpy as np
import time

numNodes = 20
case = 1
baseDir = "eval_logs/"

def bandwidth_graph():
    # for the bandwidth, you should track the average bandwidth across each second of the experiment.
    nodeNums = np.arange(numNodes) + 1

    for nodeNum in nodeNums:
        seconds = 0
        avgBandwidth = []
        window = 1 # 1 seconds
        with open(baseDir + "bandwidth_node" + str(nodeNum) + ".txt") as file:
            for line in file:
                if line == "\n":
                    line = "0 \n"# continue
                line = line[:-1] # stripping the "\n"
                seconds += window # 1
                numbers = line.split(" ")[:-1]
                for i in range(len(numbers)):
                    if numbers[i] != "":
                        numbers[i] = np.float64(numbers[i])
                if numbers != []:
                    bandwidth = np.array(numbers)

                # calculate average bandwidth = length of strings / 5 seconds
                avgBandwidth.append(np.sum(bandwidth)/window)

        # Linear scale plot
        plt.plot((np.arange(seconds/window)*window).tolist(), avgBandwidth, 'm.:', label='bandwidth')
        # plt.scatter((np.arange(seconds/window)*window).tolist(), avgBandwidth, marker='.', c='m', label='bandwidth')
        plt.ylabel('Number of Bytes Downloaded')
        plt.xlabel('Time in Seconds')
        plt.title('Bandwidth Graph Linear Case' + str(case) + '_node' + str(nodeNum))
        plt.grid()
        plt.savefig(baseDir + 'bandwidth_linear_case' + str(case) + '_node' + str(nodeNum) + '.png')
        plt.close()

        # Logarithmic scale on the y-axis
        plt.plot((np.arange(seconds/window)*window).tolist(), avgBandwidth, 'bx:', label='bandwidth')
        # plt.scatter((np.arange(seconds/window)*window).tolist(), avgBandwidth, marker='x', c='b', label='bandwidth')
        plt.yscale("log")
        plt.ylabel('Number of Bytes Downloaded')
        plt.xlabel('Time in Seconds')
        plt.title('Bandwidth Graph Log-Scaled Case' + str(case) + '_node' + str(nodeNum))
        plt.grid()
        plt.savefig(baseDir + 'bandwidth_logscale_case' + str(case) + '_node' + str(nodeNum) + '.png')
        plt.close()
    return

def reachability_graph():
    nodeNums = np.arange(numNodes) + 1

    transactionReach = dict() # format transactionID:number of files seen in

    for nodeNum in nodeNums:
        seconds = 0
        avgBandwidth = []
        window = 1 # 1 seconds
        with open(baseDir + "transactions_node" + str(nodeNum) + ".txt") as file:
            for line in file:
                line = line[:-1] # stripping the "\n"
                line = line.strip()
                currID = line.split(" ")[1]
                if currID not in transactionReach.keys():
                    transactionReach[currID] = 0
                transactionReach[currID] += 1

    # x = transactionReach.keys()

    y = []
    for key in transactionReach.keys():
        y.append(transactionReach[key])
    x = np.arange(len(y))
    xticks = transactionReach.keys()

    #"""
    # Bar Graph
    #plt.bar(x, y, label='Reachability') #, 'm.:', )
    #plt.plot(x, y, label='Reachability')
    plt.scatter(x, y, label='Reachability')
    plt.xticks(x, xticks)
    # plt.scatter((np.arange(seconds/window)*window).tolist(), avgBandwidth, marker='.', c='m', label='bandwidth')
    plt.ylabel('Number of Nodes reached')
    plt.xlabel('TransactionID')
    plt.title('Reachability Graph Case' + str(case))
    plt.grid()
    plt.savefig(baseDir + 'reachability_case' + str(case) + '.png')
    plt.close()
    """
    print(x)
    print(y)
    fig = plt.figure()
    ax = fig.add_axes([0,0,1,1])
    ax.bar(x,y)
    plt.show()
    """

    return

def propagation_delay_graph():
    return

reachability_graph()
#bandwidth_graph()
propagation_delay_graph()
