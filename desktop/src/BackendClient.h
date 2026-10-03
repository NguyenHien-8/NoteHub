#pragma once
#include <QElapsedTimer>
#include <QJsonObject>
#include <QMap>
#include <QObject>
#include <QPointer>
#include <QProcess>
#include <QTimer>
#include <functional>

class BackendClient final : public QObject {
    Q_OBJECT
  public:
    using Reply = std::function<void(const QJsonValue &, const QString &, const QString &)>;
    explicit BackendClient(QObject *parent = nullptr);
    ~BackendClient() override;
    void start(const QString &program, const QString &dataDir);
    void call(QObject *context, const QString &method, const QJsonObject &params, Reply reply);
    bool busy() const {
        return !pending.isEmpty();
    }
    bool isReady() const {
        return ready;
    }
    void stop();
  signals:
    void connected(const QJsonObject &hello);
    void failed(const QString &message);
    void busyChanged(bool busy);

  private:
    struct Pending {
        QPointer<QObject> context;
        Reply reply;
        QElapsedTimer timer;
    };
    QProcess process;
    QTimer deadlines;
    QMap<QString, Pending> pending;
    QByteArray buffer, diagnostics;
    quint64 sequence = 0;
    bool ready = false, stopping = false;
    void receive();
    void failAll(const QString &message);
};
