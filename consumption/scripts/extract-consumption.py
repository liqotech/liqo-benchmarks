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
    parser.add_argument("input_hub", help="Input file (hub)")
    parser.add_argument("input_minion", help="Input file template (minion)")
    parser.add_argument("output", help="Output file")
    parser.add_argument("--start", help="Start timestamp", type=int, required=True)
    parser.add_argument("--end", help="End timestamp", type=int, required=True)
    args = parser.parse_args()

    print(f"Processing file {args.input_hub}")
    input_hub = pd.read_csv(args.input_hub)
    metric = input_hub["metric"]

    cpu_control, cpu_kubelet = \
        process(input_hub[metric == "container_cpu_usage_seconds_total"],
                args.start, args.end, lambda p, v, t: (v - p) / t)

    memory_control, memory_kubelet = \
        process(input_hub[metric == "container_memory_working_set_bytes"],
                args.start, args.end, lambda p, v, t: v / 1e6)

    receive, _ = process(input_hub[metric == "liqo_network_receive_bytes_total"],
                         args.start, args.end, lambda p, v, t: 8 * (v - p) / t / 1e6)
    transmit, _ = process(input_hub[metric == "liqo_network_transmit_bytes_total"],
                          args.start, args.end, lambda p, v, t: 8 * (v - p) / t / 1e6)

    cpu_minion, memory_minion = list(), list()
    for idx in range(10):
        print(f"Processing file {args.input_minion % idx}")
        input_minion = pd.read_csv(args.input_minion % idx)
        metric = input_minion["metric"]

        cpu, _ = \
            process(input_minion[metric == "container_cpu_usage_seconds_total"],
                    args.start, args.end, lambda p, v, t: (v - p) / t)
        cpu_minion.append(cpu)

        memory, _ = \
            process(input_minion[metric == "container_memory_working_set_bytes"],
                    args.start, args.end, lambda p, v, t: v / 1e6)
        memory_minion.append(memory)

    cpu_minion = np.average(cpu_minion, axis=0)
    memory_minion = np.average(memory_minion, axis=0)

    time = np.arange(0, len(cpu_control), dtype=int)
    data = np.stack([time, cpu_control, cpu_kubelet, cpu_minion, memory_control,
                     memory_kubelet, memory_minion, receive, transmit], axis=1)
    columns = ["time", "cpu-control-plane", "cpu-kubelet", "cpu-minion-control-plane",
               "ram-control-plane", "ram-kubelet", "ram-minion-control-plane",
               "net-receive", "net-transmit"]
    output = pd.DataFrame(data, columns=columns)
    output.to_csv(args.output, float_format="%.3f", index=False)
