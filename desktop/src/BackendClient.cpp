#include "BackendClient.h"
#include <QFileInfo>
#include <QJsonDocument>
#include <utility>
#ifdef Q_OS_WIN
#include <windows.h>
#endif

BackendClient::BackendClient(QObject *parent) : QObject(parent) {
    process.setProcessChannelMode(QProcess::SeparateChannels);
#ifdef Q_OS_WIN
    process.setCreateProcessArgumentsModifier(
        [](QProcess::CreateProcessArguments *args) { args->flags |= CREATE_NO_WINDOW; });
#endif
    connect(&process, &QProcess::readyReadStandardOutput, this, &BackendClient::receive);
    connect(&process, &QProcess::readyReadStandardError, this, [this] {
        diagnostics += process.readAllStandardError();
        diagnostics = diagnostics.right(8192);
    });
    connect(&process, &QProcess::errorOccurred, this, [this](QProcess::ProcessError) {
        if (!stopping)
            failAll(tr("Backend process error: %1").arg(process.errorString()));
    });
    connect(&process, qOverload<int, QProcess::ExitStatus>(&QProcess::finished), this,
            [this](int code, QProcess::ExitStatus) {
                if (!stopping)
                    failAll(tr("Backend stopped (exit %1). Unsaved text is kept.\n%2")
                                .arg(code)
                                .arg(QString::fromUtf8(diagnostics)));
            });
    connect(&process, &QProcess::started, this, [this] {
        call(this, "hello", {}, [this](const QJsonValue &value, const QString &code, const QString &message) {
            if (!code.isEmpty()) {
                failAll(message);
                return;
            }
            const auto hello = value.toObject();
            if (hello["protocol"].toInt() != 1) {
                failAll(tr("Incompatible backend protocol. Deploy the matching notehub-core binary."));
                return;
            }
            ready = true;
            emit connected(hello);
        });
    });
    deadlines.setInterval(1000);
    connect(&deadlines, &QTimer::timeout, this, [this] {
        for (const auto &item : std::as_const(pending))
            if (item.timer.elapsed() > 300000) {
                failAll(tr("Backend request timed out. Its outcome may be uncertain; inspect saved notes "
                           "before retrying."));
                return;
            }
    });
    deadlines.start();
}
BackendClient::~BackendClient() {
    stop();
}
void BackendClient::start(const QString &program, const QString &dataDir) {
    if (process.state() != QProcess::NotRunning)
        return;
    stopping = false;
    ready = false;
    diagnostics.clear();
    buffer.clear();
    if (!QFileInfo(program).isFile()) {
        emit failed(
            tr("Backend not found:\n%1\nBuild and deploy notehub-core alongside NoteHub.").arg(program));
        return;
    }
    QStringList args;
    if (!dataDir.isEmpty())
        args << "--data-dir" << dataDir;
    process.start(QFileInfo(program).absoluteFilePath(), args);
}
void BackendClient::call(QObject *context, const QString &method, const QJsonObject &params, Reply reply) {
    if (stopping || process.state() != QProcess::Running || (!ready && method != "hello")) {
        reply({}, "unavailable", tr("Backend is unavailable. Your draft is retained."));
        return;
    }
    const auto id = QString::number(++sequence);
    auto record = QJsonDocument(QJsonObject{{"id", id}, {"method", method}, {"params", params}})
                      .toJson(QJsonDocument::Compact) +
                  '\n';
    if (record.size() >= (8 << 20)) {
        reply({}, "too_large", tr("This request exceeds the 8 MiB IPC limit."));
        return;
    }
    Pending item{context, std::move(reply), {}};
    item.timer.start();
    pending.insert(id, std::move(item));
    emit busyChanged(true);
    if (process.write(record) != record.size())
        failAll(tr("Could not send the backend request."));
}
void BackendClient::receive() {
    buffer += process.readAllStandardOutput();
    qsizetype end;
    while ((end = buffer.indexOf('\n')) >= 0) {
        if (end > (64 << 20)) {
            failAll(tr("Backend response exceeds the protocol limit."));
            return;
        }
        const auto line = buffer.left(end);
        buffer.remove(0, end + 1);
        QJsonParseError error;
        const auto document = QJsonDocument::fromJson(line, &error);
        if (error.error != QJsonParseError::NoError || !document.isObject()) {
            failAll(tr("Malformed backend response."));
            return;
        }
        const auto object = document.object();
        const auto id = object["id"].toString();
        if (!pending.contains(id)) {
            failAll(tr("Backend returned an unknown request identifier."));
            return;
        }
        const auto item = pending.take(id);
        const auto failure = object["error"].toObject();
        if (item.context)
            item.reply(object["result"], failure["code"].toString(), failure["message"].toString());
        emit busyChanged(busy());
    }
    if (buffer.size() > (64 << 20))
        failAll(tr("Unterminated backend response exceeds the protocol limit."));
}
void BackendClient::failAll(const QString &message) {
    if (stopping)
        return;
    ready = false;
    stopping = true;
    const auto waiting = std::exchange(pending, {});
    process.kill();
    buffer.clear();
    for (const auto &item : waiting)
        if (item.context)
            item.reply({}, "transport", message);
    emit busyChanged(false);
    emit failed(message);
}
void BackendClient::stop() {
    stopping = true;
    ready = false;
    process.closeWriteChannel();
    if (process.state() != QProcess::NotRunning && !process.waitForFinished(5000)) {
        process.kill();
        process.waitForFinished(2000);
    }
}
