import argparse

import numpy as np
import pandas as pd


def process(data, start, end, converter):
    items = dict()

    for _, row in data.iterrows():
        pod = row["pod"]
        value = row["value"]
        timestamp = row["timestamp"]

        if timestamp < start - 20 or timestamp > end + 20:
            continue

        try:
            item = items[pod]
        except KeyError:
            item = {
                "values": np.zeros(end - start),
                "last_value": value,
                "last_timestamp": timestamp
            }
            items[pod] = item

        if item["last_timestamp"] != timestamp:
            fi = max(0, item["last_timestamp"] - start)
            la = min(end - start, timestamp - start)
            if fi >= la:
                continue

            item["values"][fi:la] = converter(item["last_value"], value, la - fi)
            item["last_value"] = value
            item["last_timestamp"] = timestamp

    control, kubelet = list(), list()
    for key, value in items.items():
        if key.startswith("virtual-kubelet"):
            kubelet.append(value["values"])
        else:
            control.append(value["values"])

    return np.sum(control, axis=0), np.sum(kubelet, axis=0)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("input", help="Input file")
    parser.add_argument("output", help="Output file")
    parser.add_argument("--start", help="Start timestamp", type=int, required=True)
    parser.add_argument("--end", help="End timestamp", type=int, required=True)
    args = parser.parse_args()

    input = pd.read_csv(args.input)
    metric = input["metric"]

    cpu_control, cpu_kubelet = \
        process(input[metric == "container_cpu_usage_seconds_total"],
                args.start, args.end, lambda p, v, t: (v - p) / t)

    memory_control, memory_kubelet = \
        process(input[metric == "container_memory_working_set_bytes"],
                args.start, args.end, lambda p, v, t: v / 1e6)

    _, receive = process(input[metric == "container_network_receive_bytes_total"],
                         args.start, args.end, lambda p, v, t: 8 * (v - p) / t)
    _, transmit = process(input[metric == "container_network_transmit_bytes_total"],
                          args.start, args.end, lambda p, v, t: 8 * (v - p) / t)

    time = np.arange(0, len(cpu_control), dtype=int)
    data = np.stack([time, cpu_control, cpu_kubelet, memory_control, memory_kubelet, receive, transmit], axis=1)
    columns = ["time", "cpu-control-plane", "cpu-kubelet", "ram-control-plane", "ram-kubelet", "net-receive", "net-transmit"]
    output = pd.DataFrame(data, columns=columns)
    output.to_csv(args.output, float_format="%.3f", index=False)
