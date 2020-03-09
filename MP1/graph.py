#   Delay from the time the event is generated to the time it shows up in the centralized logger
#   The amount of bandwidth used by the centralized logger
import numpy as np
import matplotlib.pyplot as plt

def plot_delay():
    # For the delay, for each second you should plot the
    # minimum, maximum, median, and 90th percentile delay at each second
    minList = []; maximumList = []; medianList = []; ninetiethPercentileList = [];
    seconds = 0
    with open('delay.txt') as file:
        for line in file:
            if line == "\n":
                line = "0 \n"# continue
            line = line[:-2] # stripping the "\n"
            seconds += 1
            numbers = line.split(" ")
            for i in range(len(numbers)):
                numbers[i] = np.float64(numbers[i])
            delays = np.array(numbers)

            # calculate minimum, maximum, median, 90th percentile
            minList.append(np.min(delays))
            maximumList.append(np.max(delays))
            medianList.append(np.median(delays))
            ninetiethPercentileList.append(np.percentile(delays, 90))

    fig = plt.figure()
    ax1 = fig.add_subplot(111)

    ax1.plot(np.arange(seconds).tolist(), minList, marker='.', c='C1', label='minimum', linestyle='-')
    ax1.plot(np.arange(seconds).tolist(), maximumList, marker='<', c='C2', label='maximum')
    ax1.plot(np.arange(seconds).tolist(), medianList, marker='*', c='C3', label='median')
    ax1.plot(np.arange(seconds).tolist(), ninetiethPercentileList, marker='_', c='C4', label='90th percentile')
    plt.legend(loc='upper left');
    ax1.set_ylabel('Delay in Seconds')
    ax1.set_xlabel('Time in Seconds')
    ax1.set_title('Delay Graph')
    # plt.show()
    plt.savefig('delay.png')
    plt.close()
    return

def plot_bandwidth():
    # for the bandwidth, you should track the average bandwidth across each second of the experiment.
    seconds = 0
    avgBandwidth = []
    window = 5 # 5 seconds
    nodeNums = np.arange(10)

    for nodeNum in nodeNums:
        try:
            with open("bandwidth_node" + str(nodeNum) + ".txt") as file:
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
            plt.title('Bandwidth Graph Linear')
            plt.grid()
            plt.savefig('bandwidth_linear.png')
            plt.close()

            # Logarithmic scale on the y-axis
            plt.plot((np.arange(seconds/window)*window).tolist(), avgBandwidth, 'bx:', label='bandwidth')
            # plt.scatter((np.arange(seconds/window)*window).tolist(), avgBandwidth, marker='x', c='b', label='bandwidth')
            plt.yscale("log")
            plt.ylabel('Number of Bytes Downloaded')
            plt.xlabel('Time in Seconds')
            plt.title('Bandwidth Graph Log-Scaled')
            plt.grid()
            plt.savefig('bandwidth_logscale.png')
            plt.close()
        except:
            continue
    return


plot_delay()
plot_bandwidth()
