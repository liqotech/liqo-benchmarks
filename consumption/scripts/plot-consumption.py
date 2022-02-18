import argparse

import numpy as np
import pandas as pd
import matplotlib.pyplot as plt

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("inputs", help="Input file", nargs='+')
    args = parser.parse_args()

    for input in args.inputs:
        data = pd.read_csv(input)
        x = data["time"]

        fig, axs = plt.subplots(3)
        fig.suptitle(input)

        axs[0].set_title('CPU')
        axs[0].plot(x, data["cpu-control-plane"], label="control-plane")
        axs[0].plot(x, data["cpu-kubelet"], label="kubelet")
        axs[0].plot(x, data["cpu-minion-control-plane"], label="minion-control-plane")
        axs[0].legend()

        axs[1].set_title('RAM')
        axs[1].plot(x, data["ram-control-plane"], label="control-plane")
        axs[1].plot(x, data["ram-kubelet"], label="kubelet")
        axs[1].plot(x, data["ram-minion-control-plane"], label="minion-control-plane")
        axs[1].legend()

        axs[2].set_title('Network')
        axs[2].plot(x, data["net-receive"], label="receive")
        axs[2].plot(x, data["net-transmit"], label="transmit")
        axs[2].legend()

    plt.show()
