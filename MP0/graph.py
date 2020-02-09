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
            line = line[:-1] # stripping the \n
            seconds += 1
            numbers = line.split(" ")
            for i in range(len(numbers)):
                numbers[i] = float(numbers[i])
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
    ax1.plot(np.arange(seconds).tolist(), ninetiethPercentileList, marker='p', c='C4', label='90th percentile')
    plt.legend(loc='upper left');
    ax1.set_ylabel('delay in seconds')
    ax1.set_xlabel('time in seconds')
    ax1.set_title('Delay Graph')
    # plt.show()
    plt.savefig('delay.png')
    plt.close()
    return

def plot_bandwidth():
    # for the bandwidth, you should track the average bandwidth across each second of the experiment.
    seconds = 0
    avgBandwidth = []
    with open('bandwidth.txt') as file:
        for line in file:
            seconds += 1
            numbers = line.split(" ")
            for i in range(len(numbers)):
                numbers[i] = float(numbers[i])
            bandwidth = np.array(numbers)

            # calculate minimum, maximum, median, 90th percentile
            avgBandwidth.append(np.average(bandwidth))

    plt.plot(np.arange(seconds).tolist(), avgBandwidth, c='b', marker='x', label='bandwidth')
    plt.ylabel('length of strings')
    plt.xlabel('time in seconds')
    plt.title('Bandwidth Graph')
    plt.savefig('bandwidth.png')
    plt.close()
    return


plot_delay()
plot_bandwidth()
