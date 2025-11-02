export enum LogLevel {
  NONE = 0,
  ERROR = 1,
  WARN = 2,
  INFO = 3,
  DEBUG = 4,
}

export interface LogEntry {
  level: LogLevel;
  message: string;
  timestamp: number;
  context?: any;
}

export class Logger {
  private static instance: Logger;
  private logLevel: LogLevel = LogLevel.ERROR;
  private logs: LogEntry[] = [];
  private maxLogs: number = 1000;

  private constructor() {}

  static getInstance(): Logger {
    if (!Logger.instance) {
      Logger.instance = new Logger();
    }
    return Logger.instance;
  }

  setLogLevel(level: LogLevel): void {
    this.logLevel = level;
  }

  setMaxLogs(maxLogs: number): void {
    this.maxLogs = maxLogs;
    if (this.logs.length > maxLogs) {
      this.logs = this.logs.slice(-maxLogs);
    }
  }

  private shouldLog(level: LogLevel): boolean {
    return level <= this.logLevel;
  }

  private addLog(level: LogLevel, message: string, context?: any): void {
    const logEntry: LogEntry = {
      level,
      message,
      timestamp: Date.now(),
      context,
    };

    this.logs.push(logEntry);

    // Keep only the most recent logs
    if (this.logs.length > this.maxLogs) {
      this.logs = this.logs.slice(-this.maxLogs);
    }

    // Console output
    if (this.shouldLog(level)) {
      const timestamp = new Date(logEntry.timestamp).toISOString();
      const levelName = LogLevel[level];
      const logMessage = `[${timestamp}] [${levelName}] ${message}`;

      switch (level) {
        case LogLevel.ERROR:
          console.error(logMessage, context || '');
          break;
        case LogLevel.WARN:
          console.warn(logMessage, context || '');
          break;
        case LogLevel.INFO:
          console.info(logMessage, context || '');
          break;
        case LogLevel.DEBUG:
          console.debug(logMessage, context || '');
          break;
      }
    }
  }

  error(message: string, context?: any): void {
    this.addLog(LogLevel.ERROR, message, context);
  }

  warn(message: string, context?: any): void {
    this.addLog(LogLevel.WARN, message, context);
  }

  info(message: string, context?: any): void {
    this.addLog(LogLevel.INFO, message, context);
  }

  debug(message: string, context?: any): void {
    this.addLog(LogLevel.DEBUG, message, context);
  }

  getLogs(): LogEntry[] {
    return [...this.logs];
  }

  clearLogs(): void {
    this.logs = [];
  }

  getLogsByLevel(level: LogLevel): LogEntry[] {
    return this.logs.filter(log => log.level === level);
  }

  getRecentLogs(count: number = 100): LogEntry[] {
    return this.logs.slice(-count);
  }
}