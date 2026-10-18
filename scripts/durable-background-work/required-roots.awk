# SPDX-FileCopyrightText: 2026 Adrian PK
# SPDX-License-Identifier: Apache-2.0
#
# This file is part of Hatmax. See LICENSE for license terms.

# Go buffers verbose output per package. Match successful native root lines,
# then require that package's recorded families when its result is printed.
/--- SKIP: (TestLeasedRuns|TestBoundedDeliveries|TestWorkOperations)([/ (])/ { skipped = 1 }
/^--- PASS: TestLeasedRuns / { leased = 1 }
/^--- PASS: TestBoundedDeliveries / { deliveries = 1 }
/^--- PASS: TestWorkOperations / { operations = 1 }
/^ok[[:space:]]+hatmax[.]adrianpk[.]com\// {
    package = $2
    if (package == "hatmax.adrianpk.com/scheduler") {
        scheduler = 1
        if (!leased) missing = 1
    } else if (package == "hatmax.adrianpk.com/scheduler/postgres") {
        scheduler_store = 1
        if (!leased || !operations) missing = 1
    } else if (package == "hatmax.adrianpk.com/pubsub") {
        pubsub = 1
        if (!deliveries) missing = 1
    } else if (package == "hatmax.adrianpk.com/pubsub/postgres") {
        pubsub_store = 1
        if (!deliveries || !operations) missing = 1
    }
    leased = deliveries = operations = 0
}
END {
    if (missing || skipped || !scheduler || !scheduler_store || !pubsub || !pubsub_store) {
        print "Required work family absent, skipped or unsuccessful." > "/dev/stderr"
        exit 1
    }
    print "Required leased, delivery and operations families passed in all four packages."
}
