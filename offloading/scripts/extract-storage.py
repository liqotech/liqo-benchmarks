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

        return (end - start) / 1e9


def to_line(data, columns):
    cat = np.concatenate(
        [data.mean(axis=0), data.std(axis=0)])
    return pd.DataFrame([cat.tolist()], columns=columns)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("input_path", help="Input folder")
    parser.add_argument("output_file", help="Output file")
    parser.add_argument("--replicas", help="The amount of replicas", type=int, default=1)
    args = parser.parse_args()

    columns = ["avg-vanilla", "avg-liqo-local", "avg-liqo-remote", "std-vanilla", "std-liqo-local", "std-liqo-remote"]
    output = pd.DataFrame()
    for test in ["create", "attach"]:
        data = []
        for run in range(10):
            file = f"storage-{test}-vanilla-{args.replicas}-{run+1}.txt"
            vanilla = read(args.input_path, file)
            file = f"storage-{test}-liqo-local-{args.replicas}-{run+1}.txt"
            liqo_local = read(args.input_path, file)
            file = f"storage-{test}-liqo-remote-{args.replicas}-{run+1}.txt"
            liqo_remote = read(args.input_path, file)
            data.append(np.array([vanilla, liqo_local, liqo_remote]))

        output = output.append(to_line(np.stack(data), columns))

    # Adding the test column here to avoid problems with type inference.
    output.insert(0, "test", ["create", "attach"])
    output.to_csv(args.output_file, float_format="%.3f", index=False)
