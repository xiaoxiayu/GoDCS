# GoDCS

轻量级分布式任务分配框架。

1、跨平台。
2、使用脚本方式运行程序、与程序开发语言无关。
3、配置简单。

配置：

<?xml version="1.0" encoding="utf-8" ?>
<Run>
<PrjName>GoDCS Demo</PrjName>

<RemoteIP >192.168.0.1</RemoteIP>
<RemoteIP >192.168.0.1</RemoteIP>
<RemoteIP >more.and.more</RemoteIP>

<PrjPackage>/Project/Folder</PrjPackage>
<HttpPort>9999</HttpPort>

<ScriptRun>runner.py</ScriptRun>
<ScriptMonitor>runstate.py</ScriptMonitor>
<ScriptCleaner>cleaner.py</ScriptCleaner>

<ExternMonitorP>calc.exe</ExternMonitorP>

<Report>ERROR.log</Report>
<Report>SUCCESS.log</Report>
<Report>more report file</Report>

<ReportFlag>_time.txt</ReportFlag>
<ReportFlag>TestReport_</ReportFlag>

<!--  if set automatic ip will useless -->
<!-- <automatic></automatic> -->
<RecieveSpace>/report/path</RecieveSpace>

<!--  Task will run on all machines if set uniform mode -->
<!-- <TaskMode>uniform<TaskMode> -->
<Task>task0</Task>
<Task>task1</Task>
</Run>
