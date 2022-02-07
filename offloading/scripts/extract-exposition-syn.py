import argparse
import os

import numpy as np
import pandas as pd
from tabulate import tabulate


def read(path, file):
    with open(os.path.join(path, file), 'r') as input:
        start, end = 0, 0
        # Loop through the lines to extract the information of interest.
        for line in input:
            if line.startswith("Retrieved: "):
                start = int(line.removeprefix("Retrieved: "))
            if line.startswith("Connected: "):
                end = int(line.removeprefix("Connected: "))
                break

        return start, end


def to_array(start, vanilla, liqo):
    return np.array([vanilla - start, liqo - start]) / 1e9


def to_line(data, columns):
    cat = np.concatenate(
        [data.mean(axis=0), data.std(axis=0)])
    return pd.DataFrame([cat.tolist()], columns=columns)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("input_path", help="Input folder")
    parser.add_argument("output_file", help="Output file")
    args = parser.parse_args()

    columns = ["avg-vanilla", "avg-liqo", "std-vanilla", "std-liqo"]
    output = pd.DataFrame()
    data = []
    for run in range(10):
        file = f"exposition-syn-vanilla-{run+1}.txt"
        start, vanilla = read(args.input_path, file)
        file = f"exposition-syn-liqo-{run+1}.txt"
        detected, liqo = read(args.input_path, file)
        data.append(to_array(start, vanilla, liqo))

    print(tabulate(data, headers=["vanilla", "liqo"], floatfmt=".3f"))
    output = output.append(to_line(np.stack(data), columns))
    output.to_csv(args.output_file, float_format="%.3f", index=False)
