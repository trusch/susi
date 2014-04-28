susi
====

SUSI is a Universal System Interface

TCP Protocol is in json.

Structure:
{
  "id" : 123,
  "type": "publish",
  "key": "topicname",
  "authlevel" : 0,
  "payload": {"some":"data"}
}

possible types:
publish,subscribe,set,get,pop,push,enqueue,dequeue
