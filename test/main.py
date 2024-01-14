from testlib import (printLine, 
                     printNew)


import dbutils as db

import sys

import newlib
import newlib as nl

import utils.writer as wr


def main():
    printLine("This is a test")
    printNew("This is a new test")
    t = newlib.addTwo(2, 3)
    printLine(t)
    t = nl.addThree(2, 3, 5)
    printLine(t)
    t = nl.addThree(2, 3, 5)
    printLine(t)
    nl.addThreeAndPrint(2, 3, 6)
    newlib.lols()
    print(wr.innerTester(10))
    print(sys.version)
    print(db.listSql([1, 2, 3]))

if __name__ == '__main__':
    main()
