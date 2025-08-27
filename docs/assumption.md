# Assumptions

This document outlines the key assumptions made in the design and implementation of the OPA Policy Manager. These assumptions are crucial for understanding the context in which the system operates and for making informed decisions about its use and development.

## 1. Policy spec mismatch

- The spec specify that the `operator` field is optional, but does not specify what to do if it is missing. The implementation assumes that if the `operator` field is missing, it should be treated as an equality operator (`=`).
