#include "IpcClient.h"

#include <QJsonDocument>
#include <QJsonParseError>

IpcClient::IpcClient(QObject *parent) : QObject(parent) {
    process_.setProcessChannelMode(QProcess::SeparateChannels);
    connect(&process_, &QProcess::readyReadStandardOutput, this, &IpcClient::readStdout);
    connect(&process_, &QProcess::readyReadStandardError, this, &IpcClient::readStderr);
    connect(&process_, &QProcess::errorOccurred, this, [this](QProcess::ProcessError error) {
        if (stopping_) return;
        if (error == QProcess::FailedToStart) {
            emit fatal(tr("Không thể khởi động Go backend: %1").arg(process_.errorString()));
            failPending(process_.errorString());
        }
    });
    connect(&process_, qOverload<int, QProcess::ExitStatus>(&QProcess::finished), this,
            [this](int code, QProcess::ExitStatus status) {
                if (!stopping_) {
                    failPending(tr("Go backend đã dừng."));
                }
                emit backendExited(code, status);
            });
}

IpcClient::~IpcClient() { stop(); }

void IpcClient::startBackend(const QString &program, const QStringList &arguments) {
    if (process_.state() != QProcess::NotRunning) return;
    stopping_ = false;
    stdoutBuffer_.clear();
    process_.setProgram(program);
    process_.setArguments(arguments);
    process_.start();
}

qint64 IpcClient::call(const QString &method, const QJsonObject &params, Callback callback) {
    const qint64 id = nextId_++;
    if (callback) callbacks_.insert(id, std::move(callback));

    if (process_.state() == QProcess::NotRunning) {
        if (callbacks_.contains(id)) {
            auto cb = callbacks_.take(id);
            cb({}, QJsonObject{{"code", 503}, {"kind", "backend_unavailable"}, {"message", tr("Go backend chưa chạy.")}});
        }
        return id;
    }

    const QJsonObject request{{"id", id}, {"method", method}, {"params", params}};
    QByteArray payload = QJsonDocument(request).toJson(QJsonDocument::Compact);
    payload.append('\n');
    process_.write(payload);
    return id;
}

void IpcClient::stop() {
    if (process_.state() == QProcess::NotRunning) return;
    stopping_ = true;

    const QJsonObject request{{"id", nextId_++}, {"method", "app.shutdown"}, {"params", QJsonObject{}}};
    QByteArray payload = QJsonDocument(request).toJson(QJsonDocument::Compact);
    payload.append('\n');
    process_.write(payload);
    process_.waitForBytesWritten(500);
    if (!process_.waitForFinished(1800)) {
        process_.terminate();
        if (!process_.waitForFinished(800)) {
            process_.kill();
            process_.waitForFinished(500);
        }
    }
    failPending(tr("Ứng dụng đang đóng."));
}

bool IpcClient::isRunning() const { return process_.state() != QProcess::NotRunning; }

void IpcClient::readStdout() {
    stdoutBuffer_.append(process_.readAllStandardOutput());
    while (true) {
        const qsizetype newline = stdoutBuffer_.indexOf('\n');
        if (newline < 0) break;
        QByteArray line = stdoutBuffer_.left(newline).trimmed();
        stdoutBuffer_.remove(0, newline + 1);
        if (line.isEmpty()) continue;

        QJsonParseError parseError{};
        const QJsonDocument doc = QJsonDocument::fromJson(line, &parseError);
        if (parseError.error != QJsonParseError::NoError || !doc.isObject()) {
            emit backendLog(tr("IPC parse error: %1").arg(parseError.errorString()));
            continue;
        }
        handleMessage(doc.object());
    }
}

void IpcClient::readStderr() {
    const QString text = QString::fromUtf8(process_.readAllStandardError()).trimmed();
    if (!text.isEmpty()) emit backendLog(text);
}

void IpcClient::handleMessage(const QJsonObject &object) {
    const QString event = object.value("event").toString();
    if (!event.isEmpty()) {
        if (event == "ready") emit ready(object);
        else if (event == "fatal") emit fatal(object.value("message").toString());
        return;
    }

    const qint64 id = object.value("id").toVariant().toLongLong();
    if (!callbacks_.contains(id)) return;
    auto callback = callbacks_.take(id);
    callback(object.value("result"), object.value("error").toObject());
}

void IpcClient::failPending(const QString &message) {
    const auto keys = callbacks_.keys();
    for (qint64 id : keys) {
        auto callback = callbacks_.take(id);
        callback({}, QJsonObject{{"code", 503}, {"kind", "backend_unavailable"}, {"message", message}});
    }
}
