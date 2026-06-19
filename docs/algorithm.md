# Algorithm Notes

Driftwatch uses two small online detectors that are easy to explain in an interview and cheap enough to run in a service path.

## Rolling robust anomaly score

For each series, Driftwatch keeps the last `windowSize` values and computes the median plus median absolute deviation before the current value is added to the window.

The anomaly score is:

```text
abs(value - rolling_median) / max(rolling_MAD * 1.4826, epsilon)
```

The `1.4826` factor scales MAD toward the standard deviation under a normal distribution. Median/MAD is intentionally less fragile than mean/standard deviation when a backend emits one bad spike.

## EWMA baseline

The exponentially weighted moving average is reported with every detection so the operator can see whether the current value is moving the short-term baseline.

```text
ewma = alpha * value + (1 - alpha) * previous_ewma
```

## Page-Hinkley drift score

Anomaly detection is about sudden spikes. Drift detection is about a sustained level shift. Driftwatch uses a Page-Hinkley-style cumulative deviation score and emits a drift signal when the score crosses `driftThreshold`.

That makes the sample project useful for backend conversations around:

- noisy metrics and alert fatigue
- per-tenant or per-route baselines
- streaming systems where you cannot load the whole dataset
- explaining model behavior without a black-box dependency
