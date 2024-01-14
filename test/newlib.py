from testlib import printLine
import testlib as tl
import testlib as t2

import anotherlib

def addTwo(x: int, y: int) -> int:
    return x + y

def addThree(x: int, y: int, z: int) -> int:
    return x + y + z

def addFour(x: int, y: int, z: int) -> int:
    return x + y + z

def addAndPrint(x: int, y: int) -> int:
    printLine(addTwo(x, y))

def addThreeAndPrint(x: int, y: int, z:int) -> int:
    tl.printTest(addThree(x, y, z))

def addFourAndPrint(x: int, y: int, z:int, k:int) -> int:
    t2.printRandomTest(addFour(x, y, z, k))

def lols():
    anotherlib.testFunction()
    
