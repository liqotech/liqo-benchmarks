import argparse
import os

import numpy as np
import pandas as pd


def read(path, file):
    with open(os.path.join(path, file), 'r') as input:
        # Loop through the lines to extract the information of interest.
        for line in input:
            if line.startswith("Start: "):
                start = int(line.removeprefix("Start: ").split(' ')[0])
            if line.startswith("End  : "):
                end = int(line.removeprefix("End  : ").split(' ')[0])

        # Return the total time
        return np.array([end - start]) / 1e9


def to_line(pods, data, columns):
    cat = np.concatenate(
        [[pods], data.mean(axis=0), data.std(axis=0)])
    return pd.DataFrame([cat.tolist()], columns=columns)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("input_path", help="Input folder")
    parser.add_argument("output_file", help="Output file")
    args = parser.parse_args()

    output = pd.DataFrame()
    for type, max in [("vanilla", 10000), ("liqo", 10000), ("liqo-100ms", 10000), ("tensile", 10000), ("admiralty", 1000)]:
        columns = [f"count-{type}", f"avg-{type}", f"std-{type}"]

        inner = pd.DataFrame()
        for pods in [10, 100, 1000, 10000]:
            if pods > max:
                inner = inner.append(
                    to_line(pods, np.stack([np.array([0])]), columns))
                continue

            data = []
            for run in range(10):
                file = f"offloading-{type}-1-{pods}-{run+1}.txt"
                data.append(read(args.input_path, file))

            inner = inner.append(to_line(pods, np.stack(data), columns))

        output = pd.concat([output, inner], axis=1)

    output.to_csv(args.output_file, float_format="%.3f", index=False)
