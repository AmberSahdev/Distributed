import numpy as np
import matplotlib.pyplot as plt

numNodes = 8
case = "case2"
baseDir = "data/" + case

def plot_delay():
    # Combine the log files, delayLogs_node0, delayLogs_node1, ...
    # Plot the min and max delay for each transaction
    # x axis = create time, y axis = first commit time - last commit time
    nodeNums = np.arange(numNodes)
    delayDict = dict() # Fromat: {transactionID: (timeCreated,[timeCommitted])}
    transactionCreated = dict()
    transactionCommitted = dict()

    for nodeNum in nodeNums:
        with open(baseDir + "/delayLogs/delayLogs_node" + str(nodeNum) + ".txt") as file:
            for line in file:
                if line == "\n":
                    continue
                line = line[:-1] # stripping the "\n"

                fields = line.split(" ")
                time = np.float64(fields[0]); transactionID = fields[1]; type = fields[2]
                if type == "CREATED":
                    transactionCreated[transactionID] = time
                elif type == "COMMITTED":
                    try:
                        transactionCommitted[transactionID].append(time)
                    except:
                        transactionCommitted[transactionID] = [time]

    plottingData = [] # format: [time, minVal, maxVal]
    for key in transactionCreated.keys():
        try:
            minCommitTime = min(transactionCommitted[key])
            maxCommitTime = max(transactionCommitted[key])
            plottingData.append([transactionCreated[key], minCommitTime-transactionCreated[key], maxCommitTime-transactionCreated[key]])
        except:
            print("Transaction wasn't committed")

    fig = plt.figure()
    ax1 = fig.add_subplot(111)
    ax1.plot([i[0] for i in plottingData], [i[1] for i in plottingData], 'm.', label='minimum delay')
    ax1.plot([i[0] for i in plottingData], [i[2] for i in plottingData], 'bx', label='maximum delay')
    plt.legend(loc='upper right');
    ax1.set_ylabel('Delay in Seconds')
    ax1.set_xlabel('Time in nanoseconds')
    ax1.set_title('Delay Graph')
    plt.savefig(baseDir + '/delayLogs/delay_' + str(case) + '.png')
    plt.close()


def plot_bandwidth():
    # for the bandwidth, you should track the average bandwidth across each second of the experiment.
    nodeNums = np.arange(numNodes)

    for nodeNum in nodeNums:
        seconds = 0
        avgBandwidth = []
        window = 5 # 5 seconds
        with open(baseDir + "/bandwidthLogs/bandwidthLogs_node" + str(nodeNum) + ".txt") as file:
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
                avgBandwidth.append(np.sum(bandwidth))

        # Linear scale plot
        plt.plot((np.arange(seconds/window)*window).tolist(), avgBandwidth, 'm.:', label='bandwidth')
        # plt.scatter((np.arange(seconds/window)*window).tolist(), avgBandwidth, marker='.', c='m', label='bandwidth')
        plt.ylabel('Number of Bytes Downloaded')
        plt.xlabel('Time in Seconds')
        plt.title('Bandwidth Graph Linear' + case + '_node' + str(nodeNum))
        plt.grid()
        plt.savefig(baseDir + '/bandwidthLogs/bandwidth_linear_' + case + '_node' + str(nodeNum) + '.png')
        plt.close()

        # Logarithmic scale on the y-axis
        plt.plot((np.arange(seconds/window)*window).tolist(), avgBandwidth, 'bx:', label='bandwidth')
        # plt.scatter((np.arange(seconds/window)*window).tolist(), avgBandwidth, marker='x', c='b', label='bandwidth')
        plt.yscale("log")
        plt.ylabel('Number of Bytes Downloaded')
        plt.xlabel('Time in Seconds')
        plt.title('Bandwidth Graph Log-Scaled' + case + '_node' + str(nodeNum))
        plt.grid()
        plt.savefig(baseDir + '/bandwidthLogs/bandwidth_logscaled_' + case + '_node' + str(nodeNum) + '.png')
        plt.close()
    return


plot_delay()
plot_bandwidth()
