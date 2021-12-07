import argparse
import os

import numpy as np
import pandas as pd


def read(path, file):
    with open(os.path.join(path, file), 'r') as input:
        start, end = 0, 0
        # Loop through the lines to extract the information of interest.
        for line in input:
            if line.startswith("Start: "):
                start = int(line.removeprefix("Start: ").split(' ')[0])
            if line.startswith("End  : "):
                end = int(line.removeprefix("End  : ").split(' ')[0])

        return start, end


def to_array(start, vanilla, liqo):
    return np.array([vanilla - start, liqo - start]) / 1e9


def to_line(pods, data, columns):
    cat = np.concatenate(
        [[pods], data.mean(axis=0), data.std(axis=0)])
    return pd.DataFrame([cat.tolist()], columns=columns)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("input_path", help="Input folder")
    parser.add_argument("output_file", help="Output file")
    args = parser.parse_args()

    columns = ["count", "avg-vanilla", "avg-liqo", "std-vanilla", "std-liqo"]
    output = pd.DataFrame()
    for pods in [10, 100, 1000, 10000]:
        data = []
        for run in range(10):
            file = f"exposition-vanilla-1-{pods}-{run+1}.txt"
            start, vanilla = read(args.input_path, file)
            file = f"exposition-liqo-1-{pods}-{run+1}.txt"
            _, liqo = read(args.input_path, file)
            data.append(to_array(start, vanilla, liqo))

        output = output.append(to_line(pods, np.stack(data), columns))

    output.to_csv(args.output_file, float_format="%.3f", index=False)
