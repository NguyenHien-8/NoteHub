#pragma once

#include <QObject>
#include <QHash>
#include <QJsonObject>
#include <QJsonValue>
#include <QProcess>
#include <functional>

class IpcClient final : public QObject {
    Q_OBJECT
public:
    using Callback = std::function<void(const QJsonValue &, const QJsonObject &)>;

    explicit IpcClient(QObject *parent = nullptr);
    ~IpcClient() override;

    void startBackend(const QString &program, const QStringList &arguments = {});
    qint64 call(const QString &method, const QJsonObject &params, Callback callback = {});
    void stop();
    bool isRunning() const;

signals:
    void ready(const QJsonObject &info);
    void fatal(const QString &message);
    void backendLog(const QString &message);
    void backendExited(int exitCode, QProcess::ExitStatus status);

private slots:
    void readStdout();
    void readStderr();

private:
    void handleMessage(const QJsonObject &object);
    void failPending(const QString &message);

    QProcess process_;
    QByteArray stdoutBuffer_;
    qint64 nextId_ = 1;
    QHash<qint64, Callback> callbacks_;
    bool stopping_ = false;
};
