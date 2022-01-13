import argparse
import os

import numpy as np
import pandas as pd

from io import StringIO


def read(path, file):
    with open(os.path.join(path, file), 'r') as input:
        csv = ""

        # Skip the initial log lines
        for line in input:
            if line.startswith("cluster-id"):
                csv += line
                break

        # Collect the csv data
        for line in input:
            if line.isspace():
                break
            csv += line

        # Convert the output to a pandas dataframe
        return pd.read_csv(StringIO(csv), sep=',', index_col="cluster-id")


def total(data):
    return data["node-ready"] - data["peering-process-start"]


def authentication(data):
    return data["outgoing-authentication-end"] - data["peering-process-start"]


def negotiation(data):
    return data["resource-negotiation-end"] - data["resource-negotiation-start"]


def network(data):
    return data["network-setup-end"] - \
        data[["network-setup-start", "resource-negotiation-end"]].max(axis=1)


def node(data):
    start = data[["resource-negotiation-end", "network-setup-end", "virtual-kubelet-setup-start"]].max(axis=1)
    return data["node-ready"] - start


def compute(data):
    tot = list(map(lambda d: max(total(d)) / 1e9, data))

    concat = pd.concat(data)
    auth = np.mean(authentication(concat) / total(concat))
    neg = np.mean(negotiation(concat) / total(concat))
    net = np.mean(network(concat) / total(concat))
    no = np.mean(node(concat) / total(concat))
    other = 1 - auth - neg - net - no

    return np.mean(tot), np.std(tot), auth, neg, net, no, other


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("input_path", help="Input folder")
    parser.add_argument("output_file", help="Output file")
    args = parser.parse_args()

    output = ["count,total,std,authentication,negotiation,network,node,other\n"]
    for n in [1, 2, 4, 8, 16, 32, 64, 128]:
        data = []
        for run in range(10):
            file = f"peering-{n:03d}-{run+1}.txt"
            data.append(read(args.input_path, file))

        avg, std, auth, neg, net, no, other = compute(data)
        print(f"{n:03d}: {avg:6.3f}s ({std:.3f}s) - {auth:.3f} {neg:.3f}"
              f" {net:.3f} {no:.3f} {other:.3f}")
        output.append(f"{n},{avg:.3f},{std:.3f},{auth:.3f},{neg:.3f},"
                      f"{net:.3f},{no:.3f},{other:.3f}\n")

    with open(args.output_file, 'w') as file:
        file.writelines(output)
