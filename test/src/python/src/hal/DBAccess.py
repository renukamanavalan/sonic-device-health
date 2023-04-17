#! /usr/bin/env python3

from dataclasses import dataclass

/* A value >= 0 implies a valid fd */
fd = int

/* Error reported as string - More for logging */
error = str

/* 
 *  Path that describes the table in DB
 *
 *  For SONiC <DB name>/<Table name>.
 *      e.g. "CONFIG_DB/AAA" -- Points to AAA table in CONFIG-DB
 */


@dataclass
class DBData_t:
    Path:       DBPath_t
    Key:        str = None
    Data:       str = None    /* A none value implies delete/non-existing */


class DBAccess:
    """
        Init     

        Called once to initialize the DB lib.

        Return:
            Error code & Error string
    """
    def init(self) -> (int, str):
        pass


    """
        GetKeys

        This function returns keys for given path

        Input:
            path: DB path
                    
        Return:
            list of keys.
            A value of None implies non existing path
    """
    def GetKeys(self, path: DBPath_t) -> [str]:
        pass


    """
        GetEntry

        This function returns data for given path & key

        Input:
            path: DB path

            key: Key to set in given path

        Return:
            data -- Referred by the key from path
                    A None value for data.Data implies not existing.
    """
    def GetEntry(self, path: DBPath_t, key: str) -> DBData_t:
        pass


    
    """
        SetEntry

        This function sets data for given path & key
        A none value for DBData_t:Data deletes the key.

        Input:
            data: DBData_t
                  Has path, key & data to set

        Return:
            None

    """
    def SetEntry(self, data: DBData_t):
        pass


    
    """
        ModEntry

        This function updates data for given path & key

        Input:
            data: DBData_t
                  Has complete details

        Return:
            None
    """
    def ModEntry(self, path: str, key: str, data: str):
        pass


    """
        Add subscribe paths

        Input:
            path:  DBPath_t
                   Path to subscribe

            signalFd:
                >= 0 implied a valid fd. This will be signaled upon update
                < 0 implies no signaling needed.

        Return:
            None
    """
    def AddSubscribePath(self, path: DBPath_t, signalFd: int): 
        pass


    """
        GetDBUpdate
    
        Reads any updates on subscription, which can timeout with set seconds.
        A none value for DBData_t:Data implies a delete.

        Input:
            timeout:
                In seconds. The call returns with no data, on timeout
                A None value implies no timeout.
                Defaults to None

        Return:
            DBData_t  
                A none value implies timeout.
    """
    def GetDBUpdate(int timeout=None) -> DBData_t:
        pass

