# Start: 1644414340
# End  : 1644414425

import argparse

import numpy as np
import pandas as pd


def process(data, start, end, converter, timestamp_converter):
    items = dict()

    for _, row in data.iterrows():
        pod = row["pod"]
        value = row["value"]
        timestamp = timestamp_converter(row["timestamp"])

        if timestamp < start - 5 or timestamp > end + 5:
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
            fi = max(0, item["last_timestamp"] - start + 1)
            la = min(end - start, timestamp - start + 1)
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
    parser.add_argument("output", help="Output file")
    parser.add_argument("--start", help="Start timestamp", type=int, required=True)
    parser.add_argument("--end", help="End timestamp", type=int, required=True)
    args = parser.parse_args()

    print(f"Processing file {args.input_hub}")
    input_hub = pd.read_csv(args.input_hub)
    metric = input_hub["metric"]

    receive, _ = process(input_hub[metric == "liqo_network_receive_bytes_total"],
                         args.start, args.end, lambda p, v, t: 8 * (v - p) / t / 1e6,
                         lambda x: x)
    transmit, _ = process(input_hub[metric == "liqo_network_transmit_bytes_total"],
                          args.start, args.end, lambda p, v, t: 8 * (v - p) / t / 1e6,
                          lambda x: x)

    time = np.arange(0, len(receive), dtype=int)
    data = np.stack([time, receive, transmit], axis=1)
    columns = ["time", "net-receive", "net-transmit"]
    output = pd.DataFrame(data, columns=columns)
    output.to_csv(args.output, float_format="%.3f", index=False)
